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
	"github.com/BrunoReboul/ram/utilities/logging"
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
	environment                string
	iamTopicName               string
	instanceName               string
	microserviceName           string
	projectID                  string
	PubSubID                   string
	pubsubPublisherClient      *pubsub.PublisherClient
	retryTimeOutSeconds        int64
	scannerBufferSizeKiloBytes int
	splitThresholdLineNumber   int64
	step                       logging.Step
	stepStack                  logging.Steps
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
	Asset     asset         `json:"asset"`
	Window    cai.Window    `json:"window"`
	Origin    string        `json:"origin"`
	StepStack logging.Steps `json:"step_stack,omitempty"`
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
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var storageClient *storage.Client

	initID := fmt.Sprintf("%v", uuid.New())
	// err = ffo.ExploreFolder(solution.PathToFunctionCode)
	// if err != nil {
	// 	log.Printf("ffo.ExploreFolder %v", err)
	// }
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Println(logging.Entry{
			Severity:    "CRITICAL",
			Message:     "init_failed",
			Description: fmt.Sprintf("ReadUnmarshalYAML %s %v", solution.SettingsFileName, err),
			InitID:      initID,
		})
		return err
	}

	global.environment = instanceDeployment.Core.EnvironmentName
	global.instanceName = instanceDeployment.Core.InstanceName
	global.microserviceName = instanceDeployment.Core.ServiceName

	log.Println(logging.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

	global.iamTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.IAMPolicies
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.scannerBufferSizeKiloBytes = instanceDeployment.Settings.Instance.ScannerBufferSizeKiloBytes
	global.splitThresholdLineNumber = instanceDeployment.Settings.Instance.SplitThresholdLineNumber

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("storage.NewClient(ctx) %v", err),
			InitID:           initID,
		})
		return err
	}
	global.storageBucket = storageClient.Bucket(instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name)
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("pubsub.NewPublisherClient(global.ctx) %v", err),
			InitID:           initID,
		})
		return err
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, gcsEvent gcs.Event, global *Global) error {
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("pubsub_id no available metadata.FromContext: %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	global.stepStack = nil
	global.PubSubID = metadata.EventID
	parts := strings.Split(metadata.Resource.Name, "/")
	global.step = logging.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.PubSubID),
		StepTimestamp: metadata.Timestamp,
	}

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(logging.Entry{
		MicroserviceName:           global.microserviceName,
		InstanceName:               global.instanceName,
		Environment:                global.environment,
		Severity:                   "NOTICE",
		Message:                    "start",
		TriggeringPubsubID:         global.PubSubID,
		TriggeringPubsubAgeSeconds: d.Seconds(),
		TriggeringPubsubTimestamp:  &metadata.Timestamp,
		Now:                        &now,
	})

	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Println(logging.Entry{
			MicroserviceName:           global.microserviceName,
			InstanceName:               global.instanceName,
			Environment:                global.environment,
			Severity:                   "CRITICAL",
			Message:                    "noretry",
			Description:                "Pubsub message too old",
			TriggeringPubsubID:         global.PubSubID,
			TriggeringPubsubAgeSeconds: d.Seconds(),
			TriggeringPubsubTimestamp:  &metadata.Timestamp,
			Now:                        &now,
		})
		return nil
	}

	var childDumpNumber int64
	var dumpLineNumber int64
	var buffer bytes.Buffer
	var pubSubMsgNumber int64
	var startTime time.Time

	// gcsEventJSON, err := json.Marshal(gcsEvent)
	// if err != nil {
	// 	log.Println(logging.Entry{
	// 		MicroserviceName:   global.microserviceName,
	// 		InstanceName:       global.instanceName,
	// 		Environment:        global.environment,
	// 		Severity:           "WARNING",
	// 		Message:            fmt.Sprintf("json.Marshal(gcsEvent) %v", err),
	// 		TriggeringPubsubID: global.PubSubID,
	// 	})
	// } else {
	// 	log.Println(logging.Entry{
	// 		MicroserviceName:   global.microserviceName,
	// 		InstanceName:       global.instanceName,
	// 		Environment:        global.environment,
	// 		Severity:           "INFO",
	// 		Message:            "gcsEventJSON",
	// 		Description:        string(gcsEventJSON),
	// 		TriggeringPubsubID: global.PubSubID,
	// 	})
	// }

	if gcsEvent.ResourceState == "not_exists" {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "NOTICE",
			Message:            "cancel",
			Description:        fmt.Sprintf("deleted object %v", gcsEvent.Name),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	if gcsEvent.Size == "0" {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "NOTICE",
			Message:            "cancel",
			Description:        fmt.Sprintf("empty object %v", gcsEvent.Name),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	matched, _ := regexp.Match(`dumpinventory.*.dump`, []byte(gcsEvent.Name))
	if !matched {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "NOTICE",
			Message:            "cancel",
			Description:        fmt.Sprintf("not a cai dump %v", gcsEvent.Name),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	if gcsEvent.Metageneration == "1" {
		// The metageneration attribute is updated on metadata changes.
		// The on create value is 1.
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "INFO",
			Message:            fmt.Sprintf("new object tirgger %s", gcsEvent.Name),
			Description:        fmt.Sprintf("size %s", gcsEvent.Size),
			TriggeringPubsubID: global.PubSubID,
		})
	} else {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "INFO",
			Message:            fmt.Sprintf("updated object trigger %s", gcsEvent.Name),
			Description:        fmt.Sprintf("size %s", gcsEvent.Size),
			TriggeringPubsubID: global.PubSubID,
		})
	}
	storageObject := global.storageBucket.Object(gcsEvent.Name)
	storageObjectReader, err := storageObject.NewReader(global.ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("storageObject.NewReader(global.ctx) %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	defer storageObjectReader.Close()
	teeStorageObjectReader := io.TeeReader(storageObjectReader, &buffer)

	var topicList []string
	err = gps.GetTopicList(global.ctx, global.pubsubPublisherClient, global.projectID, &topicList)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("gps.GetTopicList %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}

	var gcsStep logging.Step
	parts = strings.Split(gcsEvent.Name, ".")
	if strings.Contains(parts[len(parts)-2], "child") {
		gcsStep.StepTimestamp, err = time.Parse(time.RFC3339, strings.Replace(parts[len(parts)-3], "_", ":", -1))
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "noretry",
				Description:        "time.Parse(time.RFC3339, parts[len(parts)-3]",
				TriggeringPubsubID: global.PubSubID,
			})
			return nil
		}
		gcsStep.StepID = fmt.Sprintf("%s.dump/%s", parts[0], parts[len(parts)-4])
		global.stepStack = append(global.stepStack, gcsStep)
	}

	gcsStep.StepTimestamp = gcsEvent.Updated
	gcsStep.StepID = gcsEvent.ID
	global.stepStack = append(global.stepStack, gcsStep)
	global.stepStack = append(global.stepStack, global.step)

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
		dumpLineNumber, childDumpNumber, duration, err = splitToChildDumps(buffer,
			gcsEvent.Name,
			gcsEvent.Generation,
			strings.Replace(gcsEvent.Updated.Format(time.RFC3339), ":", "_", -1),
			childDumpNumber,
			global)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "noretry",
				Description:        fmt.Sprintf("splitToChildDumps %v", err),
				TriggeringPubsubID: global.PubSubID,
			})
			return nil
		}
		childDumpNumber++
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish split to %d childDumps %s", childDumpNumber, gcsEvent.Name),
			Description:          fmt.Sprintf("dumpLineNumber %d gcsEvent.Generation %s duration %v", dumpLineNumber, gcsEvent.Generation, duration),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
	} else {
		dumpLineNumber, duration = splitToLines(buffer, global, &pubSubMsgNumber, &topicList, startTime)
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish split to %d lines %s", dumpLineNumber, gcsEvent.Name),
			Description:          fmt.Sprintf("pubSubMsgNumber %d gcsEvent.Generation %v duration %v", pubSubMsgNumber, gcsEvent.Generation, duration),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
	}
	return nil
}

func splitToChildDumps(buffer bytes.Buffer, parentDumpName string, parentGeneration string, parentTimestamp string, childDumpNumber int64, global *Global) (int64, int64, time.Duration, error) {
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
	childDumpName := strings.Replace(parentDumpName, ".dump", fmt.Sprintf(".%s.%s.child%d.dump", parentGeneration, parentTimestamp, childDumpNumber), 1)
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
					log.Println(logging.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            "fmt.Fprint(storageObjectWriter, childDumpContent)",
						Description:        fmt.Sprintf("iteration %d err %v", i, err),
						TriggeringPubsubID: global.PubSubID,
					})
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
					log.Println(logging.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            fmt.Sprintf("storageObjectWriter.Close() %s", childDumpName),
						Description:        fmt.Sprintf("iteration %d dumpLineNumber %d childDumpLineNumber %d err %v", i, dumpLineNumber, childDumpLineNumber, err),
						TriggeringPubsubID: global.PubSubID,
					})
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
			childDumpName = strings.Replace(parentDumpName, ".dump", fmt.Sprintf(".%s.%s.child%d.dump", parentGeneration, parentTimestamp, childDumpNumber), 1)
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
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            "fmt.Fprint(storageObjectWriter, childDumpContent)",
				Description:        fmt.Sprintf("iteration %d err %v", i, err),
				TriggeringPubsubID: global.PubSubID,
			})
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
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            fmt.Sprintf("storageObjectWriter.Close() %s", childDumpName),
				Description:        fmt.Sprintf("iteration %d dumpLineNumber %d childDumpLineNumber %d err %v", i, dumpLineNumber, childDumpLineNumber, err),
				TriggeringPubsubID: global.PubSubID,
			})
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
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "WARNING",
			Message:            "json.Unmarshal([]byte(dumpline), &assetLegacy)",
			Description:        fmt.Sprintf("err %v dumpline %s", err, dumpline),
			TriggeringPubsubID: global.PubSubID,
		})
	} else {
		asset := transposeAsset(assetLegacy)
		if asset.IamPolicy == nil && asset.Resource == nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            "ignored dump line: no IamPolicy object nor Resource object",
				Description:        fmt.Sprintf("dumpline %s", dumpline),
				TriggeringPubsubID: global.PubSubID,
			})
		} else {
			if asset.IamPolicy != nil {
				topicName = global.iamTopicName
			} else {
				topicName = "cai-rces-" + cai.GetAssetShortTypeName(asset.AssetType)
			}
			// log.Println("topicName", topicName)
			if err = gps.CreateTopic(global.ctx, global.pubsubPublisherClient, topicListPointer, topicName, global.projectID); err != nil {
				log.Println(logging.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "WARNING",
					Message:            fmt.Sprintf("ignored dump line: no topic to publish %s", topicName),
					Description:        fmt.Sprintf("err %v dumpline %s", err, dumpline),
					TriggeringPubsubID: global.PubSubID,
				})
			} else {
				feedMessageJSON, err := json.Marshal(getFeedMessage(asset, startTime, global))
				if err != nil {
					log.Println(logging.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "CRITICAL",
						Message:            "noretry",
						Description:        fmt.Sprintf("json.Marshal(getFeedMessage(asset, startTime)) %v", err),
						TriggeringPubsubID: global.PubSubID,
					})
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
					log.Println(logging.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            fmt.Sprintf("dump line not publihed to pubsub topic %v", err),
						TriggeringPubsubID: global.PubSubID,
					})
				}
				// log.Println(logging.Entry{
				// 	MicroserviceName:   global.microserviceName,
				// 	InstanceName:       global.instanceName,
				// 	Environment:        global.environment,
				// 	Severity:           "INFO",
				// 	Message:            fmt.Sprintf("dump line publihed to pubsub topic %s", topicName),
				// 	Description:        fmt.Sprintf("MessageIds %v feedMessageJSON %s", pubsubResponse.MessageIds, string(feedMessageJSON)),
				// 	TriggeringPubsubID: global.PubSubID,
				// })
				_ = pubsubResponse
				*pointerTopubSubMsgNumber++
			}
		}
	}
	return nil
}

func getFeedMessage(asset asset, startTime time.Time, global *Global) feedMessage {
	var feedMessage feedMessage
	feedMessage.Asset = asset
	feedMessage.Origin = "batch-export"
	feedMessage.Window.StartTime = startTime
	feedMessage.StepStack = global.stepStack
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
