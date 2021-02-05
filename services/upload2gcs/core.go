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

package upload2gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/glo"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/storage"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	assetsCollectionID            string
	bucketFolderPath              string
	bucketHandle                  *storage.BucketHandle
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	ctx                           context.Context
	environment                   string
	firestoreClient               *firestore.Client
	instanceName                  string
	microserviceName              string
	ownerLabelKeyName             string
	PubSubID                      string
	retryTimeOutSeconds           int64
	step                          glo.Step
	stepStack                     glo.Steps
	violationResolverLabelKeyName string
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset     asset      `json:"asset"`
	Window    cai.Window `json:"window"`
	Deleted   bool       `json:"deleted"`
	Origin    string     `json:"origin"`
	StepStack glo.Steps  `json:"step_stack,omitempty" firestore:"step_stack,omitempty"`
}

// asset Cloud Asset Metadata
type asset struct {
	Name                    string                 `json:"name"`
	Ancestors               []string               `json:"ancestors"`
	AncestorsDisplayName    []string               `json:"ancestorsDisplayName"`
	AncestryPath            string                 `json:"ancestryPath"`
	AncestryPathDisplayName string                 `json:"ancestryPathDisplayName"`
	AncestryPathLegacy      string                 `json:"ancestry_path"`
	AssetType               string                 `json:"assetType"`
	AssetTypeLegacy         string                 `json:"asset_type"`
	Origin                  string                 `json:"origin"`
	Owner                   string                 `json:"owner"`
	ViolationResolver       string                 `json:"violationResolver"`
	Resource                json.RawMessage        `json:"resource"`
	IamPolicy               map[string]interface{} `json:"iamPolicy"`
	IamPolicyLegacy         map[string]interface{} `json:"iam_policy"`
	ProjectID               string                 `json:"projectID"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var storageClient *storage.Client

	initID := fmt.Sprintf("%v", uuid.New())
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Println(glo.Entry{
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

	log.Println(glo.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Println(glo.Entry{
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
	// bucketHandle must be evaluated after storateClient init
	global.bucketHandle = storageClient.Bucket(instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.AssetsJSONFile.Name)

	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("cloudresourcemanager.NewService(ctx) %v", err),
			InitID:           initID,
		})
		return err
	}
	global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("cloudresourcemanagerv2.NewService(ctx) %v", err),
			InitID:           initID,
		})
		return err
	}
	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("firestore.NewClient(ctx, projectID) %v", err),
			InitID:           initID,
		})
		return err
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(glo.Entry{
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
	global.step = glo.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.PubSubID),
		StepTimestamp: metadata.Timestamp,
	}

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(glo.Entry{
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
		log.Println(glo.Entry{
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

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "NOTICE",
			Message:            "cancel",
			Description:        fmt.Sprintf("ignored pubsub message: %s", string(PubSubMessage.Data)),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Printf("pubsub_id %s ignored pubsub message: %s", global.PubSubID, string(PubSubMessage.Data))
		return nil
	}
	var feedMessage feedMessage
	err = json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal(PubSubMessage.Data, &feedMessage) %v %v", PubSubMessage.Data, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	if feedMessage.StepStack != nil {
		global.stepStack = append(feedMessage.StepStack, global.step)
	} else {
		var caiStep glo.Step
		caiStep.StepTimestamp = feedMessage.Window.StartTime
		caiStep.StepID = fmt.Sprintf("%s/%s", feedMessage.Asset.Name, caiStep.StepTimestamp.Format(time.RFC3339))
		global.stepStack = append(global.stepStack, caiStep)
		global.stepStack = append(global.stepStack, global.step)
	}
	feedMessage.StepStack = global.stepStack

	feedMessage.Asset.Origin = feedMessage.Origin
	feedMessage.Asset.AncestryPath = cai.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName, feedMessage.Asset.ProjectID = cai.BuildAncestorsDisplayName(global.ctx,
		feedMessage.Asset.Ancestors,
		global.assetsCollectionID,
		global.firestoreClient,
		global.cloudresourcemanagerService,
		global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)
	feedMessage.Asset.Owner, _ = cai.GetAssetLabelValue(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = cai.GetAssetLabelValue(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	// Legacy
	feedMessage.Asset.IamPolicyLegacy = feedMessage.Asset.IamPolicy
	feedMessage.Asset.AssetTypeLegacy = feedMessage.Asset.AssetType
	feedMessage.Asset.AncestryPathLegacy = feedMessage.Asset.AncestryPath

	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Marshal(feedMessage) %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	// log.Printf("%s", string(feedMessageJSON))
	_ = feedMessageJSON

	var objectNameSuffix string
	if feedMessage.Asset.IamPolicy == nil {
		objectNameSuffix = ".json"
	} else {
		objectNameSuffix = "_iam.json"
	}

	objectName := strings.Replace(feedMessage.Asset.Name, "/", "", 2) + objectNameSuffix
	// log.Println("objectName", objectName)
	storageObject := global.bucketHandle.Object(objectName)

	if feedMessage.Deleted == true {
		err = storageObject.Delete(global.ctx)
		if err != nil {
			if strings.Contains(err.Error(), "object doesn't exist") {
				log.Println(glo.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "WARNING",
					Message:            fmt.Sprintf("object doesn't exist, cannot delete %s", objectName),
					TriggeringPubsubID: global.PubSubID,
				})
				return nil
			}
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("storageObject.Delete(global.ctx) %s %v", objectName, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(glo.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish delete obj %s", objectName),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &metadata.Timestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
			AssetInventoryOrigin: feedMessage.Origin,
		})
	} else {
		content, err := json.MarshalIndent(feedMessage.Asset, "", "    ")
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "noretry",
				Description:        fmt.Sprintf("json.MarshalIndent(feedMessage.Asset %v", err),
				TriggeringPubsubID: global.PubSubID,
			})
			return nil
		}
		storageObjectWriter := storageObject.NewWriter(global.ctx)
		_, err = fmt.Fprint(storageObjectWriter, string(content))
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("fmt.Fprint(storageObjectWriter, string(content)) %s %v", objectName, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		err = storageObjectWriter.Close()
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("storageObjectWriter.Close() %s %v", objectName, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(glo.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish write obj %s", objectName),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &metadata.Timestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
			AssetInventoryOrigin: feedMessage.Origin,
		})
	}
	return nil
}
