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

package cai

import (
	"context"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/str"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// getDisplayName retrieive the friendly name of an ancestor
func getDisplayName(ctx context.Context,
	name string,
	collectionID string,
	firestoreClient *firestore.Client,
	cloudresourcemanagerService *cloudresourcemanager.Service,
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) (displayName string, projectID string) {
	displayName = strings.Replace(name, "/", "_", -1)
	projectID = ""
	ancestorType := strings.Split(name, "/")[0]
	knownAncestorTypes := []string{"organizations", "folders", "projects"}
	if !str.Find(knownAncestorTypes, ancestorType) {
		return displayName, projectID
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
						var dProjectIDInterface interface{} = data["projectId"]
						if dProjectID, ok := dProjectIDInterface.(string); ok {
							projectID = dProjectID
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
				projectID = resp.ProjectId
			}
		}
	}
	return displayName, projectID
}
