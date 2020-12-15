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

package publish2fs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/logging"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/BrunoReboul/ram/utilities/str"
	"github.com/google/uuid"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	collectionID        string
	ctx                 context.Context
	environment         string
	firestoreClient     *firestore.Client
	instanceName        string
	microserviceName    string
	PubSubID            string
	retryTimeOutSeconds int64
	step                logging.Step
	stepStack           logging.Steps
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset     asset         `json:"asset" firestore:"asset"`
	Window    cai.Window    `json:"window" firestore:"window"`
	Deleted   bool          `json:"deleted" firestore:"deleted"`
	Origin    string        `json:"origin" firestore:"origin"`
	StepStack logging.Steps `json:"step_stack,omitempty" firestore:"step_stack,omitempty"`
}

// Asset Cloud Asset Metadata
type asset struct {
	Name         string                 `json:"name" firestore:"name"`
	AssetType    string                 `json:"assetType" firestore:"assetType"`
	Ancestors    []string               `json:"ancestors" firestore:"ancestors"`
	AncestryPath string                 `json:"ancestryPath" firestore:"ancestryPath"`
	IamPolicy    map[string]interface{} `json:"iamPolicy" firestore:"iamPolicy,omitempty"`
	Resource     map[string]interface{} `json:"resource" firestore:"resource"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var projectID string

	initID := fmt.Sprintf("%v", uuid.New())
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

	global.collectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("firestore.NewClient %v", err),
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

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Println(logging.Entry{
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

	var feedMessage feedMessage
	err = json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Println(logging.Entry{
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
		global.stepStack = append(global.stepStack, global.step)
	}

	documentID := str.RevertSlash(feedMessage.Asset.Name)
	documentPath := global.collectionID + "/" + documentID
	if feedMessage.Deleted == true {
		_, err = global.firestoreClient.Doc(documentPath).Delete(global.ctx)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("global.firestoreClient.Doc(documentPath).Delete(global.ctx) documentPath %s %v", documentPath, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		log.Printf("pubsub_id %s DELETED document: %s", global.PubSubID, documentPath)
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish delete doc %s", documentPath),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &metadata.Timestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
	} else {
		_, err = global.firestoreClient.Doc(documentPath).Set(global.ctx, feedMessage)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("global.firestoreClient.Doc(documentPath).Set(global.ctx, feedMessage) documentPath %s %v", documentPath, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish set doc %s", documentPath),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &metadata.Timestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
	}
	return nil
}
