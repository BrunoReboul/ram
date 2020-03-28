// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package splitdump

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/BrunoReboul/ram/ram"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                      context.Context
	initFailed               bool
	retryTimeOutSeconds      int64
	iamTopicName             string
	pubSubClient             *pubsub.Client
	splitThresholdLineNumber int64
	storageBucket            *storage.BucketHandle
}

// Asset uses the new CAI feed format
type Asset struct {
	Name      string          `json:"name"`
	AssetType string          `json:"assetType"`
	Ancestors []string        `json:"ancestors"`
	IamPolicy json.RawMessage `json:"iamPolicy"`
	Resource  json.RawMessage `json:"resource"`
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset  Asset      `json:"asset"`
	Window ram.Window `json:"window"`
	Origin string     `json:"origin"`
}

// AssetLegacy uses the CAI export legacy format, not the new CAI feed format
// aka asset_type instead of assetType, iam_policy instead of iamPolicy
type AssetLegacy struct {
	Name      string          `json:"name"`
	AssetType string          `json:"asset_type"`
	Ancestors []string        `json:"ancestors"`
	IamPolicy json.RawMessage `json:"iam_policy"`
	Resource  json.RawMessage `json:"resource"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var bucketName string
	var err error
	var ok bool
	var projectID string
	var storageClient *storage.Client

	bucketName = os.Getenv("CAIEXPORTBUCKETNAME")
	global.iamTopicName = os.Getenv("IAMTOPICNAME")
	projectID = os.Getenv("GCP_PROJECT")

	log.Println("Function COLD START")
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	if global.splitThresholdLineNumber, ok = ram.GetEnvVarInt64("SPLITTHRESHOLDLINENUMBER"); !ok {
		return
	}
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("ERROR - storage.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.storageBucket = storageClient.Bucket(bucketName)
	global.pubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - pubsub.NewClient: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, gcsEvent ram.GCSEvent, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var childDumpNumber int64
	var dumpLineNumber int64
	var buffer bytes.Buffer
	var pubSubMsgNumber int64
	var startTime time.Time

	if gcsEvent.ResourceState == "not_exists" {
		log.Printf("Object %v deleted.", gcsEvent.Name)
		return nil
	}
	if gcsEvent.Size == "0" {
		log.Printf("Object %v is empty, nothing to split, ignored", gcsEvent.Name)
		return nil
	}
	matched, _ := regexp.Match(`export-.*.dump`, []byte(gcsEvent.Name))
	if !matched {
		log.Printf("Object %v is not a CAI dump", gcsEvent.Name)
		return nil
	}
	if gcsEvent.Metageneration == "1" {
		// The metageneration attribute is updated on metadata changes.
		// The on create value is 1.
		log.Printf("Object %v created, size: %v bytes\n", gcsEvent.Name, gcsEvent.Size)
	} else {
		log.Printf("Object %v updated, size: %v bytes\n", gcsEvent.Name, gcsEvent.Size)
	}
	storageObject := global.storageBucket.Object(gcsEvent.Name)
	storageObjectReader, err := storageObject.NewReader(global.ctx)
	if err != nil {
		return fmt.Errorf("storageObject.NewReader: %v", err) // RETRY
	}
	defer storageObjectReader.Close()
	teeStorageObjectReader := io.TeeReader(storageObjectReader, &buffer)

	topicList, err := ram.GetTopicList(global.ctx, global.pubSubClient)
	if err != nil {
		return fmt.Errorf("getTopicList: %v", err) // RETRY
	}

	startTime = gcsEvent.Updated
	dumpLineNumber = 0
	scanner := bufio.NewScanner(teeStorageObjectReader)
	start := time.Now()
	for scanner.Scan() {
		dumpLineNumber++
	}
	duration := time.Since(start)

	// log.Println("dumpLineNumber", dumpLineNumber, "splitThresholdLineNumber", splitThresholdLineNumber, "duration", duration)
	if dumpLineNumber > global.splitThresholdLineNumber {
		dumpLineNumber, duration, err = splitToChildDumps(buffer, gcsEvent.Name, childDumpNumber, global)
		if err != nil {
			log.Printf("ERROR - splitToChildDumps %v", err)
			return nil // NO RETRY
		}
		log.Printf("Processed %d lines, created %d childdumps files from %s generation %v duration %v\n", dumpLineNumber, childDumpNumber+1, gcsEvent.Name, gcsEvent.Generation, duration)
	} else {
		dumpLineNumber, duration = splitToLines(buffer, global, &pubSubMsgNumber, topicList, startTime)
		log.Printf("Processed %d lines %d pubsub msg from %s generation %v duration %v\n", dumpLineNumber, pubSubMsgNumber, gcsEvent.Name, gcsEvent.Generation, duration)
	}

	return nil
}

func splitToChildDumps(buffer bytes.Buffer, parentDumpName string, childDumpNumber int64, global *Global) (int64, time.Duration, error) {
	var dumpLineNumber int64
	var childDumpLineNumber int64
	var err error
	var childDumpContent string
	var i time.Duration
	var done bool
	var duration time.Duration

	scanner := bufio.NewScanner(&buffer)
	dumpLineNumber = 0
	childDumpNumber = 0

	childDumpLineNumber = 0
	childDumpContent = ""
	childDumpName := strings.Replace(parentDumpName, ".dump", fmt.Sprintf(".%d.dump", childDumpNumber), 1)
	storageObject := global.storageBucket.Object(childDumpName)
	storageObjectWriter := storageObject.NewWriter(global.ctx)
	// bufferedWriter := bufio.NewWriter(storageObjectWriter)
	start := time.Now()
	for scanner.Scan() {
		if childDumpLineNumber < global.splitThresholdLineNumber {
			childDumpContent = childDumpContent + scanner.Text() + "\n"
		} else {

			done = false
			for i = 0; i < 10; i++ {
				_, err = fmt.Fprint(storageObjectWriter, childDumpContent)
				if err != nil {
					log.Printf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					done = true
					break
				}
			}
			if !done {
				return dumpLineNumber, duration, fmt.Errorf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
			}

			done = false
			for i = 0; i < 10; i++ {
				err = storageObjectWriter.Close()
				if err != nil {
					log.Printf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					done = true
					break
				}
			}
			if !done {
				return dumpLineNumber, duration, fmt.Errorf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
			}

			childDumpNumber++
			childDumpLineNumber = 0
			childDumpName = strings.Replace(parentDumpName, ".dump", fmt.Sprintf(".%d.dump", childDumpNumber), 1)
			storageObject = global.storageBucket.Object(childDumpName)
			storageObjectWriter = storageObject.NewWriter(global.ctx)
			childDumpContent = scanner.Text() + "\n"
		}
		dumpLineNumber++
		childDumpLineNumber++
	}
	done = false
	for i = 0; i < 10; i++ {
		_, err = fmt.Fprint(storageObjectWriter, childDumpContent)
		if err != nil {
			log.Printf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			done = true
			break
		}
	}
	if !done {
		return dumpLineNumber, duration, fmt.Errorf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
	}

	done = false
	for i = 0; i < 10; i++ {
		err = storageObjectWriter.Close()
		if err != nil {
			log.Printf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			done = true
			break
		}
	}
	if !done {
		return dumpLineNumber, duration, fmt.Errorf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
	}
	duration = time.Since(start)
	return dumpLineNumber, duration, nil
}

func splitToLines(buffer bytes.Buffer, global *Global, pointerTopubSubMsgNumber *int64, topicList []string, startTime time.Time) (int64, time.Duration) {
	var dumpLineNumber int64
	scanner := bufio.NewScanner(&buffer)
	dumpLineNumber = 0
	*pointerTopubSubMsgNumber = 0
	start := time.Now()
	for scanner.Scan() {
		dumpLineNumber++
		_ = processDumpLine(scanner.Text(), global, pointerTopubSubMsgNumber, topicList, startTime)
	}
	duration := time.Since(start)
	return dumpLineNumber, duration
}

func processDumpLine(dumpline string, global *Global, pointerTopubSubMsgNumber *int64, topicList []string, startTime time.Time) error {
	var assetLegacy AssetLegacy
	var topicName string
	err := json.Unmarshal([]byte(dumpline), &assetLegacy)
	if err != nil {
		log.Println(err, dumpline)
	} else {
		asset := transposeAsset(assetLegacy)
		if asset.IamPolicy == nil && asset.Resource == nil {
			log.Println("Ignored dump line: no IamPolicy object nor Resource object", dumpline)
		} else {
			if asset.IamPolicy != nil {
				topicName = global.iamTopicName
			} else {
				topicName = "cai-rces-" + getAssetShortTypeName(asset)
			}
			// log.Println("topicName", topicName)
			if err = ram.CreateTopic(global.ctx, global.pubSubClient, topicList, topicName); err != nil {
				log.Printf("Ignored dump line: no topic %s to publish %s %v", topicName, dumpline, err)
			} else {
				feedMessageJSON, err := json.Marshal(getFeedMessage(asset, startTime))
				if err != nil {
					log.Println("Error json.Marshal", err)
					return err
				}
				publishRequest := ram.PublishRequest{Topic: topicName}
				pubSubMessage := &pubsub.Message{
					Data: feedMessageJSON,
				}
				_, err = global.pubSubClient.Topic(publishRequest.Topic).Publish(global.ctx, pubSubMessage).Get(global.ctx)
				if err != nil {
					log.Printf("ERROR pubSubClient.Topic(publishRequest.Topic).Publish: %v", err)
				}
				*pointerTopubSubMsgNumber++
			}
		}
	}
	return nil
}

func getFeedMessage(asset Asset, startTime time.Time) FeedMessage {
	var feedMessage FeedMessage
	feedMessage.Asset = asset
	feedMessage.Origin = "batch-export"
	feedMessage.Window.StartTime = startTime
	return feedMessage
}

func transposeAsset(assetLegacy AssetLegacy) Asset {
	var asset Asset
	asset.Name = assetLegacy.Name
	asset.AssetType = assetLegacy.AssetType
	asset.IamPolicy = assetLegacy.IamPolicy
	asset.Resource = assetLegacy.Resource
	asset.Ancestors = assetLegacy.Ancestors
	return asset
}

func getAssetShortTypeName(asset Asset) string {
	var serviceName string
	tmpArr := strings.Split(asset.AssetType, "/")
	assetTypeName := tmpArr[len(tmpArr)-1]
	serviceTypeName := tmpArr[0]
	switch serviceTypeName {
	case "rbac.authorization.k8s.io":
		serviceName = "k8srbac"
	case "extensions.k8s.io":
		serviceName = "k8sextensions"
	case "networking.k8s.io":
		serviceName = "k8snetworking"
	default:
		tmpArr := strings.Split(asset.AssetType, ".")
		serviceName = tmpArr[0]
	}
	assetShortTypeName := serviceName + "-" + assetTypeName
	return assetShortTypeName
}
