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
	firestoreClient     *firestore.Client
	pubsubID            string
	retryTimeOutSeconds int64
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset   asset      `json:"asset" firestore:"asset"`
	Window  cai.Window `json:"window" firestore:"window"`
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
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx
	var instanceDeployment InstanceDeployment
	var projectID string

	logEntryPrefix := fmt.Sprintf("init_id %s", uuid.New())
	log.Printf("%s function COLD START", logEntryPrefix)
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("%s ReadUnmarshalYAML %s %v", logEntryPrefix, solution.SettingsFileName, err)
	}
	global.collectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("%s firestore.NewClient: %v", logEntryPrefix, err)
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return fmt.Errorf("pubsub_id no available REDO_ON_TRANSIENT metadata.FromContext: %v", err)
	}
	global.pubsubID = metadata.EventID

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Printf("pubsub_id %s NORETRY_ERROR pubsub message too old. max age sec %d now %v event timestamp %s", global.pubsubID, global.retryTimeOutSeconds, now, metadata.Timestamp)
		return nil
	}

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Printf("pubsub_id %s ignored pubsub message: %s", global.pubsubID, string(PubSubMessage.Data))
		return nil
	}

	var feedMessage feedMessage
	err = json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR pubSubMessage.Data cannot be UnMarshalled as a feed %s %s", global.pubsubID, string(PubSubMessage.Data), err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	// log.Printf("%v", feedMessage)

	documentID := str.RevertSlash(feedMessage.Asset.Name)
	documentPath := global.collectionID + "/" + documentID
	if feedMessage.Deleted == true {
		_, err = global.firestoreClient.Doc(documentPath).Delete(global.ctx)
		if err != nil {
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT error when deleting %s %v", global.pubsubID, documentPath, err)
		}
		log.Printf("pubsub_id %s DELETED document: %s", global.pubsubID, documentPath)
	} else {
		_, err = global.firestoreClient.Doc(documentPath).Set(global.ctx, feedMessage)
		if err != nil {
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT firestoreClient.Doc(documentPath).Set: %s %v", global.pubsubID, documentPath, err) // RETRY
		}
		log.Printf("pubsub_id %s SET document: %s", global.pubsubID, documentPath)
	}
	return nil
}
