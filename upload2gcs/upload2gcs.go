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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/helper"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/storage"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
	assetsCollectionID            string
	bucketFolderPath              string
	bucketName                    string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	firestoreClient               *firestore.Client
	initFailed                    bool
	ownerLabelKeyName             string
	projectID                     string
	retryTimeOutSeconds           int64
	storageBucket                 *storage.BucketHandle
	storageClient                 *storage.Client
	violationResolverLabelKeyName string
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset   Asset  `json:"asset"`
	Window  Window `json:"window"`
	Deleted bool   `json:"deleted"`
	Origin  string `json:"origin"`
}

// Window Cloud Asset Inventory feed message time window
type Window struct {
	StartTime time.Time `json:"startTime"`
}

// Asset Cloud Asset Metadata
type Asset struct {
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
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	global.assetsCollectionID = os.Getenv("ASSETSCOLLECTIONID")
	global.bucketName = os.Getenv("BUCKETNAME")
	global.ownerLabelKeyName = os.Getenv("OWNERLABELKEYNAME")
	global.projectID = os.Getenv("GCP_PROJECT")
	global.violationResolverLabelKeyName = os.Getenv("VIOLATIONRESOLVERLABELKEYNAME")

	log.Println("Function COLD START")
	// err is pre-declared to avoid shadowing client.
	var err error
	global.retryTimeOutSeconds, err = strconv.ParseInt(os.Getenv("RETRYTIMEOUTSECONDS"), 10, 64)
	if err != nil {
		log.Printf("ERROR - Env variable RETRYTIMEOUTSECONDS cannot be converted to int64: %v", err)
		global.initFailed = true
		return
	}
	global.storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("ERROR - storage.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.storageBucket = global.storageClient.Bucket(global.bucketName)
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - cloudresourcemanager.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - cloudresourcemanagerv2.NewService: %v", err)
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
	if global.initFailed {
		log.Println("ERROR - init function failed")
		return nil // NO RETRY
	}

	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return fmt.Errorf("metadata.FromContext: %v", err) // RETRY
	}

	// Ignore events that are too old.
	expiration := metadata.Timestamp.Add(time.Duration(global.retryTimeOutSeconds) * time.Second)
	if time.Now().After(expiration) {
		log.Printf("ERROR - too many retries for expired event '%q'", metadata.EventID)
		return nil // NO MORE RETRY
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var feedMessage FeedMessage
	err = json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal: %v", err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	feedMessage.Asset.Origin = feedMessage.Origin
	feedMessage.Asset.AncestryPath = helper.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = helper.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = helper.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)
	feedMessage.Asset.Owner, _ = helper.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = helper.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	// Legacy
	feedMessage.Asset.IamPolicyLegacy = feedMessage.Asset.IamPolicy
	feedMessage.Asset.AssetTypeLegacy = feedMessage.Asset.AssetType
	feedMessage.Asset.AncestryPathLegacy = feedMessage.Asset.AncestryPath

	// log.Printf("%v", feedMessage)

	var objectNameSuffix string
	if feedMessage.Asset.IamPolicy == nil {
		objectNameSuffix = ".json"
	} else {
		objectNameSuffix = "_iam.json"
	}

	objectName := strings.Replace(feedMessage.Asset.Name, "/", "", 2) + objectNameSuffix
	// log.Println("objectName", objectName)
	storageObject := global.storageBucket.Object(objectName)

	if feedMessage.Deleted == true {
		err = storageObject.Delete(global.ctx)
		if err != nil {
			if strings.Contains(err.Error(), "object doesn't exist") {
				log.Printf("ERROR - object doesn't exist, cannot delete %s %v", objectName, err)
				return nil // NO RETRY
			}
			return fmt.Errorf("Error when deleting %s %v", objectName, err) // RETRY
		}
		log.Printf("DELETED object: %s", objectName)
	} else {
		content, err := json.MarshalIndent(feedMessage.Asset, "", "    ")
		if err != nil {
			log.Printf("ERROR - json.Marshal(feedMessage.Asset): %v", err)
			return nil // NO RETRY
		}
		storageObjectWriter := storageObject.NewWriter(global.ctx)
		_, err = fmt.Fprint(storageObjectWriter, string(content))
		if err != nil {
			return fmt.Errorf("fmt.Fprint(storageObjectWriter, string(content)): %s %v", objectName, err) // RETRY
		}
		err = storageObjectWriter.Close()
		if err != nil {
			return fmt.Errorf("storageObjectWriter.Close(): %s %v", objectName, err) // RETRY
		}
		log.Printf("WRITE object: %s", objectName)
	}
	return nil
}
