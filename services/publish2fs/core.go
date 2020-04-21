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

	"github.com/BrunoReboul/ram/utilities/ram"

	"cloud.google.com/go/firestore"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                 context.Context
	initFailed          bool
	retryTimeOutSeconds int64
	collectionID        string
	firestoreClient     *firestore.Client
	solution            ram.SolutionSettings
	service             ServiceSettings
	instance            InstanceSettings
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset   asset      `json:"asset" firestore:"asset"`
	Window  ram.Window `json:"window" firestore:"window"`
	Deleted bool       `json:"deleted" firestore:"deleted"`
	Origin  string     `json:"origin" firestore:"origin"`
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
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	// var ok bool
	var projectID string
	var settings GoGCFDeployment

	log.Println("Function COLD START")
	err = ram.ReadUnmarshalYAML(fmt.Sprintf("./%s", ram.SettingsFileName), &settings)
	if err != nil {
		log.Printf("ERROR - ReadUnmarshalYAML %s %v", ram.SettingsFileName, err)
		global.initFailed = true
		return
	}
	global.collectionID = settings.Settings.Solution.Hosting.FireStore.CollectionIDs.Assets
	global.retryTimeOutSeconds = settings.Settings.Service.GCF.RetryTimeOutSeconds
	projectID = settings.Settings.Solution.Hosting.ProjectID

	// global.collectionID = os.Getenv("COLLECTION_ID")
	// projectID = os.Getenv("GCP_PROJECT")
	// if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
	// 	return
	// }

	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - firestore.NewClient: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var feedMessage feedMessage
	err := json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal: %v", err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	// log.Printf("%v", feedMessage)

	documentID := ram.RevertSlash(feedMessage.Asset.Name)
	documentPath := global.collectionID + "/" + documentID
	if feedMessage.Deleted == true {
		_, err = global.firestoreClient.Doc(documentPath).Delete(global.ctx)
		if err != nil {
			return fmt.Errorf("Error when deleting %s %v", documentPath, err) // RETRY
		}
		log.Printf("DELETED document: %s", documentPath)
	} else {
		_, err = global.firestoreClient.Doc(documentPath).Set(global.ctx, feedMessage)
		if err != nil {
			return fmt.Errorf("firestoreClient.Doc(documentPath).Set: %s %v", documentPath, err) // RETRY
		}
		log.Printf("SET document: %s", documentPath)
	}
	return nil
}
