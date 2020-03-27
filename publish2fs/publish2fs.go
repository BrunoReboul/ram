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
	"os"
	"strconv"

	"github.com/BrunoReboul/ram/helper"

	"cloud.google.com/go/firestore"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                 context.Context
	initFailed          bool
	retryTimeOutSeconds int64
	projectID           string
	collectionID        string
	firestoreClient     *firestore.Client
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset   Asset         `json:"asset" firestore:"asset"`
	Window  helper.Window `json:"window" firestore:"window"`
	Deleted bool          `json:"deleted" firestore:"deleted"`
	Origin  string        `json:"origin" firestore:"origin"`
}

// Asset Cloud Asset Metadata
type Asset struct {
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
	global.projectID = os.Getenv("GCP_PROJECT")
	global.collectionID = os.Getenv("COLLECTION_ID")
	log.Println("Function COLD START")
	// err is pre-declared to avoid shadowing client.
	var err error
	global.retryTimeOutSeconds, err = strconv.ParseInt(os.Getenv("RETRYTIMEOUTSECONDS"), 10, 64)
	if err != nil {
		log.Printf("ERROR - Env variable RETRYTIMEOUTSECONDS cannot be converted to int64: %v", err)
		global.initFailed = true
		return
	}
	global.firestoreClient, err = firestore.NewClient(ctx, global.projectID)
	if err != nil {
		log.Printf("ERROR - firestore.NewClient: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage helper.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := helper.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var feedMessage FeedMessage
	err := json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal: %v", err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	// log.Printf("%v", feedMessage)

	documentID := helper.RevertSlash(feedMessage.Asset.Name)
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
