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
	firestoreClient               *firestore.Client
	ownerLabelKeyName             string
	pubsubID                      string
	retryTimeOutSeconds           int64
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
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx
	var instanceDeployment InstanceDeployment
	var storageClient *storage.Client

	logEntryPrefix := fmt.Sprintf("init_id %s", uuid.New())
	log.Printf("%s function COLD START", logEntryPrefix)
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("%s ReadUnmarshalYAML %s %v", logEntryPrefix, solution.SettingsFileName, err)
	}

	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("%s storage.NewClient: %v", logEntryPrefix, err)
	}
	// bucketHandle must be evaluated after storateClient init
	global.bucketHandle = storageClient.Bucket(instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.AssetsJSONFile.Name)

	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		return fmt.Errorf("%s cloudresourcemanager.NewService: %v", logEntryPrefix, err)
	}
	global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(ctx)
	if err != nil {
		return fmt.Errorf("%s cloudresourcemanagerv2.NewService: %v", logEntryPrefix, err)
	}
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
		log.Printf("pubsub_id %s NORETRY_ERROR PubSubMessage.Data cannot be UnMarshalled as a feed %s %v", global.pubsubID, string(PubSubMessage.Data), err)
		return nil
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	feedMessage.Asset.Origin = feedMessage.Origin
	feedMessage.Asset.AncestryPath = cai.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = cai.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)
	feedMessage.Asset.Owner, _ = cai.GetAssetLabelValue(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = cai.GetAssetLabelValue(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	// Legacy
	feedMessage.Asset.IamPolicyLegacy = feedMessage.Asset.IamPolicy
	feedMessage.Asset.AssetTypeLegacy = feedMessage.Asset.AssetType
	feedMessage.Asset.AncestryPathLegacy = feedMessage.Asset.AncestryPath

	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Marshal(feedMessage)", global.pubsubID)
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
				log.Printf("pubsub_id %s NORETRY_ERROR object doesn't exist, cannot delete %s %v", global.pubsubID, objectName, err)
				return nil
			}
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT error when deleting %s %v", global.pubsubID, objectName, err)
		}
		log.Printf("pubsub_id %s DELETED object: %s", global.pubsubID, objectName)
	} else {
		content, err := json.MarshalIndent(feedMessage.Asset, "", "    ")
		if err != nil {
			log.Printf("pubsub_id %s NORETRY_ERROR json.Marshal(feedMessage.Asset): %v", global.pubsubID, err)
			return nil
		}
		storageObjectWriter := storageObject.NewWriter(global.ctx)
		_, err = fmt.Fprint(storageObjectWriter, string(content))
		if err != nil {
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT fmt.Fprint(storageObjectWriter, string(content)): %s %v", global.pubsubID, objectName, err)
		}
		err = storageObjectWriter.Close()
		if err != nil {
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT storageObjectWriter.Close(): %s %v", global.pubsubID, objectName, err)
		}
		log.Printf("pubsub_id %s WRITE object: %s", global.pubsubID, objectName)
	}
	return nil
}
