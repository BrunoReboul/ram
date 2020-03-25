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

package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iterator"
)

// PublishRequest Pub/sub
type PublishRequest struct {
	Topic string `json:"topic"`
}

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// BuildAncestorsDisplayName build a slice of Ancestor friendly name fron a slice of ancestors
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

// CreateTopic check if a topic already exist, if not create it
func CreateTopic(ctx context.Context, pubSubClient *pubsub.Client, topicList []string, topicName string) error {
	if Find(topicList, topicName) {
		return nil
	}
	// refresh topic list
	topicList, err := GetTopicList(ctx, pubSubClient)
	if err != nil {
		return fmt.Errorf("getTopicList: %v", err)
	}
	if Find(topicList, topicName) {
		return nil
	}
	topic, err := pubSubClient.CreateTopic(ctx, topicName)
	if err != nil {
		matched, _ := regexp.Match(`.*AlreadyExists.*`, []byte(err.Error()))
		if !matched {
			return fmt.Errorf("pubSubClient.CreateTopic: %v", err)
		}
	}
	log.Println("Created topic:", topic.ID())
	return nil
}

// Find a string in a slice of string. Return true when found else false
func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// fireStoreGetDoc check if a document exist with retries
func fireStoreGetDoc(ctx context.Context, firestoreClient *firestore.Client, documentPath string, retriesNumber time.Duration) (*firestore.DocumentSnapshot, bool) {
	var documentSnap *firestore.DocumentSnapshot
	var err error
	var i time.Duration
	for i = 0; i < retriesNumber; i++ {
		documentSnap, err = firestoreClient.Doc(documentPath).Get(ctx)
		if err != nil {
			log.Printf("ERROR - iteration %d firestoreClient.Doc(documentPath).Get(ctx) %v", i, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			return documentSnap, documentSnap.Exists()
		}
	}
	return documentSnap, false
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

// getDisplayName retrieive the friendly name of an ancestor
func getDisplayName(ctx context.Context, name string, collectionID string, firestoreClient *firestore.Client, cloudresourcemanagerService *cloudresourcemanager.Service, cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) string {
	var displayName = "unknown"
	ancestorType := strings.Split(name, "/")[0]
	knownAncestorTypes := []string{"organizations", "folders", "projects"}
	if !Find(knownAncestorTypes, ancestorType) {
		return displayName
	}
	documentID := "//cloudresourcemanager.googleapis.com/" + name
	documentID = RevertSlash(documentID)
	documentPath := collectionID + "/" + documentID
	// log.Printf("documentPath:%s", documentPath)
	// documentSnap, err := firestoreClient.Doc(documentPath).Get(ctx)
	documentSnap, found := fireStoreGetDoc(ctx, firestoreClient, documentPath, 10)
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

// GetTopicList retreive the list of existing pubsub topics
func GetTopicList(ctx context.Context, pubSubClient *pubsub.Client) ([]string, error) {
	var topicList []string
	topicsIterator := pubSubClient.Topics(ctx)
	for {
		topic, err := topicsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return topicList, fmt.Errorf("topicsIterator.Next: %v", err)
		}
		topicList = append(topicList, topic.ID())
	}
	return topicList, nil
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

// RevertSlash replace slash / by back slash \
func RevertSlash(txt string) string {
	return strings.Replace(txt, "/", "\\", -1)
}
