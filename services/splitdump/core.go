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
	"regexp"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gcs"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"

	"cloud.google.com/go/functions/metadata"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/storage"
	"github.com/BrunoReboul/ram/utilities/gps"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                        context.Context
	iamTopicName               string
	projectID                  string
	PubSubID                   string
	pubsubPublisherClient      *pubsub.PublisherClient
	retryTimeOutSeconds        int64
	scannerBufferSizeKiloBytes int
	splitThresholdLineNumber   int64
	storageBucket              *storage.BucketHandle
}

// asset uses the new CAI feed format
type asset struct {
	Name      string          `json:"name"`
	AssetType string          `json:"assetType"`
	Ancestors []string        `json:"ancestors"`
	IamPolicy json.RawMessage `json:"iamPolicy"`
	Resource  json.RawMessage `json:"resource"`
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset  asset      `json:"asset"`
	Window cai.Window `json:"window"`
	Origin string     `json:"origin"`
}

// assetLegacy uses the CAI export legacy format, not the new CAI feed format
// aka asset_type instead of assetType, iam_policy instead of iamPolicy
type assetLegacy struct {
	Name      string          `json:"name"`
	AssetType string          `json:"asset_type"`
	Ancestors []string        `json:"ancestors"`
	IamPolicy json.RawMessage `json:"iam_policy"`
	Resource  json.RawMessage `json:"resource"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx
	var instanceDeployment InstanceDeployment
	var storageClient *storage.Client

	logEntryPrefix := fmt.Sprintf("init_id %s", uuid.New())
	log.Printf("%s function COLD START", logEntryPrefix)
	// err = ffo.ExploreFolder(solution.PathToFunctionCode)
	// if err != nil {
	// 	log.Printf("ffo.ExploreFolder %v", err)
	// }
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("%s ReadUnmarshalYAML %s %v", logEntryPrefix, solution.SettingsFileName, err)
	}

	global.iamTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.IAMPolicies
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.scannerBufferSizeKiloBytes = instanceDeployment.Settings.Instance.ScannerBufferSizeKiloBytes
	global.splitThresholdLineNumber = instanceDeployment.Settings.Instance.SplitThresholdLineNumber

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("%s storage.NewClient: %v", logEntryPrefix, err)
	}
	global.storageBucket = storageClient.Bucket(instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name)
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		return fmt.Errorf("%s global.pubsubPublisherClient: %v", logEntryPrefix, err)
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, gcsEvent gcs.Event, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return fmt.Errorf("pubsub_id no available REDO_ON_TRANSIENT metadata.FromContext: %v", err)
	}
	global.PubSubID = metadata.EventID

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Printf("pubsub_id %s NORETRY_ERROR pubsub message too old. max age sec %d now %v event timestamp %s", global.PubSubID, global.retryTimeOutSeconds, now, metadata.Timestamp)
		return nil
	}

	var childDumpNumber int64
	var dumpLineNumber int64
	var buffer bytes.Buffer
	var pubSubMsgNumber int64
	var startTime time.Time

	if gcsEvent.ResourceState == "not_exists" {
		log.Printf("pubsub_id %s object %v deleted.", global.PubSubID, gcsEvent.Name)
		return nil
	}
	if gcsEvent.Size == "0" {
		log.Printf("pubsub_id %s object %v is empty, nothing to split, ignored", global.PubSubID, gcsEvent.Name)
		return nil
	}
	matched, _ := regexp.Match(`dumpinventory.*.dump`, []byte(gcsEvent.Name))
	if !matched {
		log.Printf("pubsub_id %s object %v is not a CAI dump", global.PubSubID, gcsEvent.Name)
		return nil
	}
	if gcsEvent.Metageneration == "1" {
		// The metageneration attribute is updated on metadata changes.
		// The on create value is 1.
		log.Printf("pubsub_id %s object %v created, size: %v bytes\n", global.PubSubID, gcsEvent.Name, gcsEvent.Size)
	} else {
		log.Printf("pubsub_id %s object %v updated, size: %v bytes\n", global.PubSubID, gcsEvent.Name, gcsEvent.Size)
	}
	storageObject := global.storageBucket.Object(gcsEvent.Name)
	storageObjectReader, err := storageObject.NewReader(global.ctx)
	if err != nil {
		return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT storageObject.NewReader: %v", global.PubSubID, err)
	}
	defer storageObjectReader.Close()
	teeStorageObjectReader := io.TeeReader(storageObjectReader, &buffer)

	var topicList []string
	err = gps.GetTopicList(global.ctx, global.pubsubPublisherClient, global.projectID, &topicList)
	if err != nil {
		return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT getTopicList: %v", global.PubSubID, err)
	}

	startTime = gcsEvent.Updated
	dumpLineNumber = 0
	scanner := bufio.NewScanner(teeStorageObjectReader)
	scannerBuffer := make([]byte, global.scannerBufferSizeKiloBytes*1024)
	scanner.Buffer(scannerBuffer, global.scannerBufferSizeKiloBytes*1024)
	start := time.Now()
	for scanner.Scan() {
		dumpLineNumber++
	}
	duration := time.Since(start)

	// log.Println("dumpLineNumber", dumpLineNumber, "splitThresholdLineNumber", splitThresholdLineNumber, "duration", duration)
	if dumpLineNumber > global.splitThresholdLineNumber {
		dumpLineNumber, childDumpNumber, duration, err = splitToChildDumps(buffer, gcsEvent.Name, childDumpNumber, global)
		if err != nil {
			log.Printf("pubsub_id %s NORETRY_ERROR splitToChildDumps %v", global.PubSubID, err)
			return nil
		}
		childDumpNumber++
		log.Printf("pubsub_id %s processed %d lines, created %d childdumps files from %s generation %v duration %v\n", global.PubSubID, dumpLineNumber, childDumpNumber, gcsEvent.Name, gcsEvent.Generation, duration)
	} else {
		dumpLineNumber, duration = splitToLines(buffer, global, &pubSubMsgNumber, &topicList, startTime)
		log.Printf("pubsub_id %s processed %d lines %d pubsub msg from %s generation %v duration %v\n", global.PubSubID, dumpLineNumber, pubSubMsgNumber, gcsEvent.Name, gcsEvent.Generation, duration)
	}
	return nil
}

func splitToChildDumps(buffer bytes.Buffer, parentDumpName string, childDumpNumber int64, global *Global) (int64, int64, time.Duration, error) {
	var dumpLineNumber int64
	var childDumpLineNumber int64
	var err error
	var childDumpContent string
	var i time.Duration
	var done bool
	var duration time.Duration

	scanner := bufio.NewScanner(&buffer)
	scannerBuffer := make([]byte, global.scannerBufferSizeKiloBytes*1024)
	scanner.Buffer(scannerBuffer, global.scannerBufferSizeKiloBytes*1024)

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
					log.Printf("pubsub_id %s error iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", global.PubSubID, i, err)
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					done = true
					break
				}
			}
			if !done {
				return dumpLineNumber, childDumpNumber, duration, fmt.Errorf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
			}

			done = false
			for i = 0; i < 10; i++ {
				err = storageObjectWriter.Close()
				if err != nil {
					log.Printf("pubsub_id %s error iteration %v storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", global.PubSubID, i, childDumpName, dumpLineNumber, childDumpLineNumber, err)
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					done = true
					break
				}
			}
			if !done {
				return dumpLineNumber, childDumpNumber, duration, fmt.Errorf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
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
			log.Printf("pubsub_id %s error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", global.PubSubID, i, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			done = true
			break
		}
	}
	if !done {
		return dumpLineNumber, childDumpNumber, duration, fmt.Errorf("Error - iteration %v fmt.Fprint(storageObjectWriter, childDumpContent): %v", i, err)
	}

	done = false
	for i = 0; i < 10; i++ {
		err = storageObjectWriter.Close()
		if err != nil {
			log.Printf("pubsub_id %s error - iteration %v storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", global.PubSubID, i, childDumpName, dumpLineNumber, childDumpLineNumber, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			done = true
			break
		}
	}
	if !done {
		return dumpLineNumber, childDumpNumber, duration, fmt.Errorf("storageObjectWriter.Close %s dumpLineNumber %d childDumpLineNumber %d %v", childDumpName, dumpLineNumber, childDumpLineNumber, err)
	}
	duration = time.Since(start)
	return dumpLineNumber, childDumpNumber, duration, nil
}

func splitToLines(buffer bytes.Buffer, global *Global, pointerTopubSubMsgNumber *int64, topicListPointer *[]string, startTime time.Time) (int64, time.Duration) {
	var dumpLineNumber int64
	scanner := bufio.NewScanner(&buffer)
	scannerBuffer := make([]byte, global.scannerBufferSizeKiloBytes*1024)
	scanner.Buffer(scannerBuffer, global.scannerBufferSizeKiloBytes*1024)

	dumpLineNumber = 0
	*pointerTopubSubMsgNumber = 0
	start := time.Now()
	for scanner.Scan() {
		dumpLineNumber++
		_ = processDumpLine(scanner.Text(), global, pointerTopubSubMsgNumber, topicListPointer, startTime)
	}
	duration := time.Since(start)
	return dumpLineNumber, duration
}

func processDumpLine(dumpline string, global *Global, pointerTopubSubMsgNumber *int64, topicListPointer *[]string, startTime time.Time) error {
	var assetLegacy assetLegacy
	var topicName string
	err := json.Unmarshal([]byte(dumpline), &assetLegacy)
	if err != nil {
		log.Println(err, dumpline)
	} else {
		asset := transposeAsset(assetLegacy)
		if asset.IamPolicy == nil && asset.Resource == nil {
			log.Printf("pubsub_id %s ignored dump line: no IamPolicy object nor Resource object %s", global.PubSubID, dumpline)
		} else {
			if asset.IamPolicy != nil {
				topicName = global.iamTopicName
			} else {
				topicName = "cai-rces-" + cai.GetAssetShortTypeName(asset.AssetType)
			}
			// log.Println("topicName", topicName)
			if err = gps.CreateTopic(global.ctx, global.pubsubPublisherClient, topicListPointer, topicName, global.projectID); err != nil {
				log.Printf("pubsub_id %s ignored dump line: no topic %s to publish %s %v", global.PubSubID, topicName, dumpline, err)
			} else {
				feedMessageJSON, err := json.Marshal(getFeedMessage(asset, startTime))
				if err != nil {
					log.Printf("pubsub_id %s NORETRY_ERROR json.Marshal %v", global.PubSubID, err)
					return err
				}
				var pubSubMessage pubsubpb.PubsubMessage
				pubSubMessage.Data = feedMessageJSON

				var pubsubMessages []*pubsubpb.PubsubMessage
				pubsubMessages = append(pubsubMessages, &pubSubMessage)

				var publishRequest pubsubpb.PublishRequest
				publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, topicName)
				publishRequest.Messages = pubsubMessages

				pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
				if err != nil {
					log.Printf("pubsub_id %s NORETRY_ERROR global.pubsubPublisherClient.Publish: %v", global.PubSubID, err)
				}
				// log.Printf("Published to pubsub topic %s ids %v %s", topicName, pubsubResponse.MessageIds, string(feedMessageJSON))
				_ = pubsubResponse
				*pointerTopubSubMsgNumber++
			}
		}
	}
	return nil
}

func getFeedMessage(asset asset, startTime time.Time) feedMessage {
	var feedMessage feedMessage
	feedMessage.Asset = asset
	feedMessage.Origin = "batch-export"
	feedMessage.Window.StartTime = startTime
	return feedMessage
}

func transposeAsset(assetLegacy assetLegacy) asset {
	var asset asset
	asset.Name = assetLegacy.Name
	asset.AssetType = assetLegacy.AssetType
	asset.IamPolicy = assetLegacy.IamPolicy
	asset.Resource = assetLegacy.Resource
	asset.Ancestors = assetLegacy.Ancestors
	return asset
}
