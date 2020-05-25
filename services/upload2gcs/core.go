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

	"github.com/BrunoReboul/ram/utilities/ram"
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
	Window  ram.Window `json:"window"`
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
	err = ram.ReadUnmarshalYAML(fmt.Sprintf("./%s", ram.SettingsFileName), &instanceDeployment)
	if err != nil {
		log.Printf("ERROR - ReadUnmarshalYAML %s %v", ram.SettingsFileName, err)
		global.initFailed = true
		return
	}

	bucketName := instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.AssetsJSONFile.Name
	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	location := instanceDeployment.Core.SolutionSettings.Hosting.GCF.Region
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("ERROR - storage.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.bucketHandle, err = getBucketHandle(global.ctx, bucketName, projectID, location, storageClient)
	if err != nil {
		log.Printf("ERROR - getBucketHandle: %v", err)
		global.initFailed = true
		return
	}
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
}

func getBucketHandle(ctx context.Context, bucketName string, projectID string, location string, storageClient *storage.Client) (bucketHandle *storage.BucketHandle, err error) {
	bucketHandle = storageClient.Bucket(bucketName)
	bucketAttrs, err := bucketHandle.Attrs(ctx)
	if err != nil {
		if err == storage.ErrBucketNotExist {
			var bucketTocreateAttrs storage.BucketAttrs
			bucketTocreateAttrs.Location = location
			bucketTocreateAttrs.StorageClass = "STANDARD"
			bucketTocreateAttrs.Labels = map[string]string{"name": strings.ToLower(bucketName)}

			err = bucketHandle.Create(ctx, projectID, &bucketTocreateAttrs)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					bucketAttrs, err = bucketHandle.Attrs(ctx)
					if err != nil {
						return nil, err
					}
				}
				return nil, fmt.Errorf("bucketHandle.Create %v", err)
			}
			log.Printf("Created bucket %s", bucketName)
			return bucketHandle, nil
		}
	}
	needToUpdate := false
	if bucketAttrs.Labels != nil {
		if value, ok := bucketAttrs.Labels["name"]; ok {
			if value != bucketAttrs.Name {
				needToUpdate = true
			}
		} else {
			needToUpdate = true
		}
	} else {
		needToUpdate = true
	}
	if needToUpdate {
		var bucketAttrsToUpdate storage.BucketAttrsToUpdate
		bucketAttrsToUpdate.SetLabel("name", strings.ToLower(bucketName))
		bucketAttrs, err = bucketHandle.Update(ctx, bucketAttrsToUpdate)
		if err != nil {
			return nil, fmt.Errorf("ERROR when updating bucket labels %v", err)
		}
		log.Printf("Update bucket labels %s", bucketName)
	}
	return bucketHandle, err
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
	feedMessage.Asset.Origin = feedMessage.Origin
	feedMessage.Asset.AncestryPath = ram.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = ram.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = ram.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)
	feedMessage.Asset.Owner, _ = ram.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = ram.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

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