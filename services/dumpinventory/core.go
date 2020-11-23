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

	asset "cloud.google.com/go/asset/apiv1"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                 context.Context
	retryTimeOutSeconds int64
	assetClient         *asset.Client
	request             *assetpb.ExportAssetsRequest
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx

	var instanceDeployment InstanceDeployment

	log.Println("Function COLD START")
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("ERROR - ReadUnmarshalYAML %s %v", solution.SettingsFileName, err)
	}

	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds

	var gcsDestinationURI assetpb.GcsDestination_Uri
	gcsDestinationURI.Uri = fmt.Sprintf("gs://%s/%s.dump",
		instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name,
		os.Getenv("K_SERVICE"))

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
		return fmt.Errorf("ERROR - unsupported content type: %s", instanceDeployment.Settings.Instance.CAI.ContentType)
	}

	global.request.Parent = instanceDeployment.Settings.Instance.CAI.Parent
	global.request.AssetTypes = instanceDeployment.Settings.Instance.CAI.AssetTypes
	global.request.OutputConfig = &outputConfig

	global.assetClient, err = asset.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("ERROR - asset.NewClient: %v", err)
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := gcf.IntialRetryCheck(ctxEvent, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	operation, err := global.assetClient.ExportAssets(global.ctx, global.request)
	if err != nil {
		return fmt.Errorf("assetClient.ExportAssets: %v", err) // RETRY
	}
	log.Printf("gcloud asset operations describe %s %v", operation.Name(), global.request)
	// do NOT wait for response to save function execution time, and avoid function timeout
	return nil
}
