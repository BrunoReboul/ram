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

package dumpinventory

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/glo"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	assetClient         *asset.Client
	ctx                 context.Context
	dumpName            string
	environment         string
	firestoreClient     *firestore.Client
	instanceName        string
	microserviceName    string
	PubSubID            string
	projectID           string
	request             *assetpb.ExportAssetsRequest
	retryTimeOutSeconds int64
	step                glo.Step
	stepStack           glo.Steps
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment

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

	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	global.dumpName = fmt.Sprintf("%s/%s.dump",
		instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name,
		os.Getenv("K_SERVICE"))

	var gcsDestinationURI assetpb.GcsDestination_Uri
	gcsDestinationURI.Uri = fmt.Sprintf("gs://%s", global.dumpName)

	var gcsDestination assetpb.GcsDestination
	gcsDestination.ObjectUri = &gcsDestinationURI

	var outputConfigGCSDestination assetpb.OutputConfig_GcsDestination
	outputConfigGCSDestination.GcsDestination = &gcsDestination

	var outputConfig assetpb.OutputConfig
	outputConfig.Destination = &outputConfigGCSDestination

	global.request = &assetpb.ExportAssetsRequest{}
	switch instanceDeployment.Settings.Instance.CAI.ContentType {
	case "RESOURCE":
		global.request.ContentType = assetpb.ContentType_RESOURCE
	case "IAM_POLICY":
		global.request.ContentType = assetpb.ContentType_IAM_POLICY
	default:
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("unsupported content type: %s", instanceDeployment.Settings.Instance.CAI.ContentType),
			InitID:           initID,
		})
		return err
	}

	global.request.Parent = instanceDeployment.Settings.Instance.CAI.Parent
	global.request.AssetTypes = instanceDeployment.Settings.Instance.CAI.AssetTypes
	global.request.OutputConfig = &outputConfig

	global.assetClient, err = asset.NewClient(ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("asset.NewClient(ctx) %v", err),
			InitID:           initID,
		})
		return err
	}
	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
	if err != nil {
		log.Println(glo.Entry{
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
	global.stepStack = append(global.stepStack, global.step)

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

	operation, err := global.assetClient.ExportAssets(global.ctx, global.request)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("global.assetClient.ExportAssets(global.ctx, global.request) %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	// do NOT wait for response to save function execution time, and avoid function timeout
	log.Println(glo.Entry{
		MicroserviceName:   global.microserviceName,
		InstanceName:       global.instanceName,
		Environment:        global.environment,
		Severity:           "INFO",
		Message:            fmt.Sprintf("gcloud asset operations describe %s", operation.Name()),
		TriggeringPubsubID: global.PubSubID,
	})
	err = gfs.RecordDump(global.ctx,
		global.dumpName,
		global.firestoreClient,
		global.stepStack,
		global.microserviceName,
		global.instanceName,
		global.environment,
		global.PubSubID,
		5)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("recordDump %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	now = time.Now()
	latency := now.Sub(metadata.Timestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(glo.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              fmt.Sprintf("finish export request to %s", global.dumpName),
		Description:          fmt.Sprintf("operationName %s request %v", operation.Name(), global.request),
		Now:                  &now,
		TriggeringPubsubID:   global.PubSubID,
		OriginEventTimestamp: &metadata.Timestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
		AssetInventoryOrigin: "batch-export",
	})
	return nil
}
