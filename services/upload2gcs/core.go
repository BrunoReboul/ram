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

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
	assetsCollectionID            string
	bucketFolderPath              string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	firestoreClient               *firestore.Client
	initFailed                    bool
	ownerLabelKeyName             string
	retryTimeOutSeconds           int64
	bucketHandle                  *storage.BucketHandle
	violationResolverLabelKeyName string
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset   asset      `json:"asset"`
	Window  cai.Window `json:"window"`
	Deleted bool       `json:"deleted"`
	Origin  string     `json:"origin"`
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
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	var instanceDeployment InstanceDeployment
	var storageClient *storage.Client

	log.Println("Function COLD START")
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Printf("ERROR - ReadUnmarshalYAML %s %v", solution.SettingsFileName, err)
		global.initFailed = true
		return
	}

	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("ERROR - storage.NewClient: %v", err)
		global.initFailed = true
		return
	}
	// bucketHandle must be evaluated after storateClient init
	global.bucketHandle = storageClient.Bucket(instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.AssetsJSONFile.Name)

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
	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - firestore.NewClient: %v", err)
		global.initFailed = true
		return
	}
	// log.Println("Done COLD START")

}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := gcf.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Printf("Ignored pubsub message: %s", string(PubSubMessage.Data))
		return nil // NO RETRY
	}

	var feedMessage feedMessage
	err := json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - PubSubMessage.Data cannot be UnMarshalled as a feed %s %s", string(PubSubMessage.Data), err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	feedMessage.Asset.Origin = feedMessage.Origin
	feedMessage.Asset.AncestryPath = cai.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = cai.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)
	feedMessage.Asset.Owner, _ = cai.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = cai.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	// Legacy
	feedMessage.Asset.IamPolicyLegacy = feedMessage.Asset.IamPolicy
	feedMessage.Asset.AssetTypeLegacy = feedMessage.Asset.AssetType
	feedMessage.Asset.AncestryPathLegacy = feedMessage.Asset.AncestryPath

	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Println("ERROR - json.Marshal(feedMessage)")
		return nil // NO RETRY
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
