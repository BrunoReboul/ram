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

// Package ram avoid code redundancy by grouping types and functions used by other ram packages
package ram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/str"

	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// BuildAncestorsDisplayName build a slice of Ancestor friendly name from a slice of ancestors
func BuildAncestorsDisplayName(ctx context.Context, ancestors []string, collectionID string, firestoreClient *firestore.Client, cloudresourcemanagerService *cloudresourcemanager.Service, cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) []string {
	cnt := len(ancestors)
	ancestorsDisplayName := make([]string, len(ancestors))
	for idx := 0; idx < cnt; idx++ {
		ancestorsDisplayName[idx] = getDisplayName(ctx, ancestors[idx], collectionID, firestoreClient, cloudresourcemanagerService, cloudresourcemanagerServiceV2)
	}
	return ancestorsDisplayName
}

// BuildAncestryPath build a path from a slice of ancestors
func BuildAncestryPath(ancestors []string) string {
	cnt := len(ancestors)
	revAncestors := make([]string, len(ancestors))
	for idx := 0; idx < cnt; idx++ {
		revAncestors[cnt-idx-1] = ancestors[idx]
	}
	var ancestryPath string
	ancestryPath = makeCompatible(strings.Join(revAncestors, "/"))
	return ancestryPath
}

// GetAssetContact retrieve owner of resolver contact from asset labels and parent labels
func GetAssetContact(contactRole string, resourceJSON json.RawMessage) (string, error) {
	var contact string
	var resource struct {
		Data struct {
			Labels map[string]string
		}
	}
	err := json.Unmarshal(resourceJSON, &resource)
	if err != nil {
		return "", err
	}
	if resource.Data.Labels != nil {
		if labelValue, ok := resource.Data.Labels[contactRole]; ok {
			contact = labelValue
		}
	}
	return contact, nil
}

// GetByteSet return a set of lenght contiguous bytes starting at bytes
func GetByteSet(start byte, length int) []byte {
	byteSet := make([]byte, length)
	for i := range byteSet {
		byteSet[i] = start + byte(i)
	}
	return byteSet
}

// getDisplayName retrieive the friendly name of an ancestor
func getDisplayName(ctx context.Context, name string, collectionID string, firestoreClient *firestore.Client, cloudresourcemanagerService *cloudresourcemanager.Service, cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) string {
	var displayName = "unknown"
	ancestorType := strings.Split(name, "/")[0]
	knownAncestorTypes := []string{"organizations", "folders", "projects"}
	if !str.Find(knownAncestorTypes, ancestorType) {
		return displayName
	}
	documentID := "//cloudresourcemanager.googleapis.com/" + name
	documentID = str.RevertSlash(documentID)
	documentPath := collectionID + "/" + documentID
	// log.Printf("documentPath:%s", documentPath)
	// documentSnap, err := firestoreClient.Doc(documentPath).Get(ctx)
	documentSnap, found := gfs.GetDoc(ctx, firestoreClient, documentPath, 10)
	if found {
		assetMap := documentSnap.Data()
		// log.Println(assetMap)
		var assetInterface interface{} = assetMap["asset"]
		if asset, ok := assetInterface.(map[string]interface{}); ok {
			var resourceInterface interface{} = asset["resource"]
			if resource, ok := resourceInterface.(map[string]interface{}); ok {
				var dataInterface interface{} = resource["data"]
				if data, ok := dataInterface.(map[string]interface{}); ok {
					switch ancestorType {
					case "organizations":
						var dNameInterface interface{} = data["displayName"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					case "folders":
						var dNameInterface interface{} = data["displayName"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					case "projects":
						var dNameInterface interface{} = data["name"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					}
				}
			}
		}
		// log.Printf("name %s displayName %s", name, displayName)
	} else {
		log.Printf("WARNING - Not found in firestore %s", documentPath)
		//try resourcemamager API
		switch strings.Split(name, "/")[0] {
		case "organizations":
			resp, err := cloudresourcemanagerService.Organizations.Get(name).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerService.Organizations.Get %v", err)
			} else {
				displayName = resp.DisplayName
			}
		case "folders":
			resp, err := cloudresourcemanagerServiceV2.Folders.Get(name).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerServiceV2.Folders.Get %v", err)
			} else {
				displayName = resp.DisplayName
			}
		case "projects":
			resp, err := cloudresourcemanagerService.Projects.Get(strings.Split(name, "/")[1]).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerService.Projects.Get %v", err)
			} else {
				displayName = resp.Name
			}
		}
	}
	return displayName
}

// GetPublishCallResult func to be used in go routine to scale pubsub event publish
func GetPublishCallResult(ctx context.Context, publishResult *pubsub.PublishResult, waitgroup *sync.WaitGroup, msgInfo string, pubSubErrNumber *uint64, pubSubMsgNumber *uint64, logEventEveryXPubSubMsg uint64) {
	defer waitgroup.Done()
	id, err := publishResult.Get(ctx)
	if err != nil {
		log.Printf("ERROR count %d on %s: %v", atomic.AddUint64(pubSubErrNumber, 1), msgInfo, err)
		return
	}
	msgNumber := atomic.AddUint64(pubSubMsgNumber, 1)
	if msgNumber%logEventEveryXPubSubMsg == 0 {
		// No retry on pubsub publish as already implemented in the GO client
		log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
	}
	// log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
}

// IntialRetryCheck performs intitial controls
// 1) return true and metadata when controls are passed
// 2) return false when controls failed:
// - 2a) with an error to retry the cloud function entry point function
// - 2b) with nil to stop the cloud function entry point function
func IntialRetryCheck(ctxEvent context.Context, initFailed bool, retryTimeOutSeconds int64) (bool, *metadata.Metadata, error) {
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return false, metadata, fmt.Errorf("metadata.FromContext: %v", err) // RETRY
	}
	if initFailed {
		log.Println("ERROR - init function failed")
		return false, metadata, nil // NO RETRY
	}

	// Ignore events that are too old.
	expiration := metadata.Timestamp.Add(time.Duration(retryTimeOutSeconds) * time.Second)
	if time.Now().After(expiration) {
		log.Printf("ERROR - too many retries for expired event '%q'", metadata.EventID)
		return false, metadata, nil // NO MORE RETRY
	}
	return true, metadata, nil
}

// makeCompatible update a GCP asset ancestryPath to make it compatible with former Policy Library REGO rules
func makeCompatible(path string) string {
	path = strings.Replace(path, "organizations", "organization", -1)
	path = strings.Replace(path, "folders", "folder", -1)
	path = strings.Replace(path, "projects", "project", -1)
	return path
}

// PrintEnptyInterfaceType discover the type below an empty interface
func PrintEnptyInterfaceType(value interface{}, valueName string) error {
	switch t := value.(type) {
	default:
		log.Printf("type %T for value named: %s\n", t, valueName)
	}
	return nil
}
