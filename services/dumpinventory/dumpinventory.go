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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	asset "cloud.google.com/go/asset/apiv1"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"

	"github.com/BrunoReboul/ram/utilities/ram"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                 context.Context
	initFailed          bool
	retryTimeOutSeconds int64
	assetClient         *asset.Client
	request             *assetpb.ExportAssetsRequest
}

// Settings the structure of the export.json setting file
type Settings struct {
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	ContentType string   `json:"contentType"`
	AssetTypes  []string `json:"asset_types"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var assetDumpFileName string
	var caiExportBucketName string
	var contentType assetpb.ContentType
	var err error
	var functionName string
	var ok bool
	var settings Settings
	var settingsFileName string

	caiExportBucketName = os.Getenv("CAIEXPORTBUCKETNAME")
	functionName = os.Getenv("FUNCTION_NAME")
	settingsFileName = os.Getenv("SETTINGSFILENAME")
	assetDumpFileName = fmt.Sprintf("gs://%s/%s.dump", caiExportBucketName, functionName)

	log.Println("Function COLD START")
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	settingsFileContent, err := ioutil.ReadFile(settingsFileName)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadFile: %v", err)
		global.initFailed = true
		return
	}
	err = json.Unmarshal(settingsFileContent, &settings)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(settingsFileContent, &settings): %v", err)
		global.initFailed = true
		return
	}
	parent := fmt.Sprintf("%s/%s", settings.Type, settings.ID)
	switch settings.ContentType {
	case "CONTENT_TYPE_UNSPECIFIED":
		contentType = 0
	case "RESOURCE":
		contentType = 1
	case "IAM_POLICY":
		contentType = 2
	case "ORG_POLICY":
		contentType = 4
	case "ACCESS_POLICY":
		contentType = 5
	}
	// services are initialized with context.Background() because it should
	// persist between function invocations.
	global.assetClient, err = asset.NewClient(ctx)
	if err != nil {
		log.Printf("ERROR - asset.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.request = &assetpb.ExportAssetsRequest{
		Parent:      parent,
		AssetTypes:  settings.AssetTypes,
		ContentType: contentType,
		OutputConfig: &assetpb.OutputConfig{
			Destination: &assetpb.OutputConfig_GcsDestination{
				GcsDestination: &assetpb.GcsDestination{
					ObjectUri: &assetpb.GcsDestination_Uri{
						Uri: string(assetDumpFileName),
					},
				},
			},
		},
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
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
