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

package convertlog2feed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	admin "google.golang.org/api/admin/directory/v1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Severity (string) incompatible with both pakage:
// loggingpb "google.golang.org/genproto/googleapis/logging/v2" got erro json: cannot unmarshal string into Go struct field LogEntry.severity of type ltype.LogSeverity
// "cloud.google.com/go/logging" got error json: cannot unmarshal string into Go struct field Entry.Severity of type logging.Severity
type logEntry struct {
	InsertID         string    `json:"insertId"`
	Timestamp        time.Time `json:"timestamp"`
	ReceiveTimestamp time.Time `json:"receiveTimestamp"`
	Resource         struct {
		Type   string            `json:"type"`
		Labels map[string]string `json:"labels"`
	} `json:"resource"`
	ProtoPayload json.RawMessage `json:"protoPayload"`
}

// https://developers.google.com/admin-sdk/reports/v1/reference/activity-ref-appendix-a/admin-event-names
type protoPayload struct {
	ServiceName  string `json:"serviceName"`
	MethodName   string `json:"methodName"`
	ResourceName string `json:"resourceName"`
	Metadata     struct {
		Events []event `json:"event"`
	} `json:"metadata"`
}

type event struct {
	EventName string          `json:"eventName"`
	EventType string          `json:"eventType"`
	Parameter json.RawMessage `json:"parameter"`
}

// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#GROUP_SETTINGS
type groupSettingsParameters []struct {
	Label string `json:"label"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	GCIGroupMembersTopicName    string
	GCIGroupSettingsTopicName   string
	cloudresourcemanagerService *cloudresourcemanager.Service
	collectionID                string
	ctx                         context.Context
	dirAdminService             *admin.Service
	directoryCustomerID         string
	firestoreClient             *firestore.Client
	groupsSettingsService       *groupssettings.Service
	initFailed                  bool
	logEntry                    logEntry
	organizationID              string
	projectID                   string
	pubsubPublisherClient       *pubsub.PublisherClient
	retryTimeOutSeconds         int64
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	var instanceDeployment InstanceDeployment
	var clientOption option.ClientOption
	var ok bool

	log.Println("Function COLD START")
	err = ram.ReadUnmarshalYAML(fmt.Sprintf("./%s", ram.SettingsFileName), &instanceDeployment)
	if err != nil {
		log.Printf("ERROR - ReadUnmarshalYAML %s %v", ram.SettingsFileName, err)
		global.initFailed = true
		return
	}

	gciAdminUserToImpersonate := instanceDeployment.Settings.Instance.GCI.SuperAdminEmail
	global.collectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.GCIGroupMembersTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	global.GCIGroupSettingsTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := "./" + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := os.Getenv("FUNCTION_IDENTITY")

	if clientOption, ok = ram.GetClientOptionAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, global.projectID, gciAdminUserToImpersonate, []string{"https://www.googleapis.com/auth/apps.groups.settings", "https://www.googleapis.com/auth/admin.directory.group.readonly"}); !ok {
		return
	}
	global.dirAdminService, err = admin.NewService(ctx, clientOption)
	if err != nil {
		log.Printf("ERROR - admin.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.groupsSettingsService, err = groupssettings.NewService(ctx, clientOption)
	if err != nil {
		log.Printf("ERROR - groupssettings.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		log.Printf("ERROR - global.pubsubPublisherClient: %v", err)
		global.initFailed = true
		return
	}
	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
	if err != nil {
		log.Printf("ERROR - firestore.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - cloudresourcemanager.NewService: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	ok, metadata, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds)
	if !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)
	// log.Printf("PubSubMessage.Data %s", PubSubMessage.Data)
	_ = metadata

	err = json.Unmarshal(PubSubMessage.Data, &global.logEntry)
	if err != nil {
		log.Printf("ERROR json.Unmarshal logentry %v", err)
		return nil
	}

	switch global.logEntry.Resource.Type {
	case "audited_resource":
		switch global.logEntry.Resource.Labels["service"] {
		case "admin.googleapis.com":
			return convertAdminActivityEvent(global)
		default:
			log.Printf("Unmanaged  global.logEntry.Resource.Labels service  %s", global.logEntry.Resource.Labels["service"])
			return nil
		}
	default:
		log.Printf("Unmanaged logEntry.Resource.Type %s", global.logEntry.Resource.Type)
		return nil
	}
}

// https://developers.google.com/admin-sdk/reports/v1/reference/activity-ref-appendix-a/admin-event-names
func convertAdminActivityEvent(global *Global) (err error) {
	var protoPayload protoPayload

	err = json.Unmarshal(global.logEntry.ProtoPayload, &protoPayload)
	if err != nil {
		log.Printf("ERROR json.Unmarshal protoPaylaod %v", err)
		return nil
	}

	parts := strings.Split(protoPayload.ResourceName, "/")
	global.organizationID = parts[1]
	// log.Printf("global.organizationID", global.organizationID)
	err = getCustomerID(global)
	if err != nil {
		return err // retry
	}
	if global.directoryCustomerID = "" {
		log.Println("ERROR directoryCustomerID not found")
		return nil
	}

	for _, event := range protoPayload.Metadata.Events {
		switch event.EventType {
		case "GROUP_SETTINGS":
			return convertGroupSettings(&event, global)
		default:
			log.Printf("Unmanaged event.EventType %s", event.EventType)
			return nil
		}
	}
	return nil
}

func convertGroupSettings(event *event, global *Global) (err error) {
	var parameters groupSettingsParameters
	err = json.Unmarshal(event.Parameter, &parameters)
	if err != nil {
		log.Printf("ERROR json.Unmarshal groupSettingsParameters %v", err)
		return nil
	}
	switch event.EventName {
	case "CHANGE_GROUP_SETTING":
		for _, parameter := range parameters {
			if parameter.Name == "GROUP_EMAIL" {
				return publishGroupSettings(parameter.Value, false, global)
			}
		}
		log.Printf("ERROR CHANGE_GROUP_SETTING expected parameter GROUP_EMAIL not found, insertId %s", global.logEntry.InsertID)
		return nil
	default:
		log.Printf("Unmanaged event.EventName %s", event.EventName)
		return nil
	}
}

func publishGroupSettings(groupEmail string, isDeleted bool, global *Global) (err error) {
	var feedMessageGroupSettings ram.FeedMessageGroupSettings
	feedMessageGroupSettings.Window.StartTime = global.logEntry.Timestamp
	feedMessageGroupSettings.Origin = "real-time-log-export"
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Deleted = isDeleted

	var groupID string
	if !isDeleted {
		groupSettings, err := global.groupsSettingsService.Groups.Get(groupEmail).Do()
		if err != nil {
			return fmt.Errorf("groupsSettingsService.Groups.Get: %v", err) // RETRY
		}
		feedMessageGroupSettings.Asset.Resource = groupSettings

		// groupKey: he value can be the group's email address, group alias, or the unique group ID.
		// https://developers.google.com/admin-sdk/directory/v1/reference/groups/get
		group, err := global.dirAdminService.Groups.Get(groupEmail).Context(global.ctx).Do()
		if err != nil {
			return fmt.Errorf("dirAdminService.Groups.Get %v", err)
		}
		groupID = group.Id
	} else {
		// WIP get group from firestore cache
		groupID = ""
	}

	feedMessageGroupSettings.Asset.Ancestors = []string{fmt.Sprintf("directories/%s", global.directoryCustomerID)}
	feedMessageGroupSettings.Asset.Name = fmt.Sprintf("//directories/%s/groups/%s/groupSettings", global.directoryCustomerID, groupID)

	feedMessageGroupSettingsJSON, err := json.Marshal(feedMessageGroupSettings)
	if err != nil {
		log.Println("ERROR - json.Marshal(feedMessageGroupSettings)")
		return nil // NO RETRY
	}

	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageGroupSettingsJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, global.GCIGroupSettingsTopicName)
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		return fmt.Errorf("global.pubsubPublisherClient.Publish: %v", err) // RETRY
	}
	log.Printf("Group %s %s settings published to pubsub topic %s ids %v %s",
		feedMessageGroupSettings.Asset.Name,
		feedMessageGroupSettings.Asset.Resource.Email,
		global.GCIGroupSettingsTopicName,
		pubsubResponse.MessageIds,
		string(feedMessageGroupSettingsJSON))
	return nil
}

func getCustomerID(global *Global) (err error) {
	documentID := fmt.Sprintf("//cloudresourcemanager.googleapis.com/organizations/%s", global.organizationID)
	documentID = ram.RevertSlash(documentID)
	documentPath := global.collectionID + "/" + documentID
	// log.Printf("documentPath %s", documentPath)
	documentSnap, found := ram.FireStoreGetDoc(global.ctx, global.firestoreClient, documentPath, 10)
	if found {
		log.Printf("Found firestore document %s", documentPath)
	
		assetMap := documentSnap.Data()
		assetMapJSON, err := json.Marshal(assetMap)
		if err != nil {
			log.Println("ERROR - json.Marshal(assetMap)")
			return nil // NO RETRY
		}	
		log.Printf("%s", string(assetMapJSON))

		var assetInterface interface{} = assetMap["asset"]
		if asset, ok := assetInterface.(map[string]interface{}); ok {
			var resourceInterface interface{} = asset["resource"]
			if resource, ok := resourceInterface.(map[string]interface{}); ok {
				var dataInterface interface{} = resource["data"]
				if data, ok := dataInterface.(map[string]interface{}); ok {
					var ownerInterface interface{} = data["owner"]
					if owner, ok := ownerInterface.(map[string]interface{}); ok {
						var directoryCustomerIDInterface interface{} = owner["directoryCustomerId"]
						if directoryCustomerID, ok := directoryCustomerIDInterface.(string); ok {
							global.directoryCustomerID = directoryCustomerID
							return nil
						}
					}

				}
			}
		}
	} else {
		log.Printf("WARNING - Not found in firestore %s", documentPath)
		//try resourcemamager API
		resp, err := global.cloudresourcemanagerService.Organizations.Get(global.organizationID).Context(global.ctx).Do()
		if err != nil {
			log.Printf("WARNING - cloudresourcemanagerService.Organizations.Get %v", err)
		} else {
			global.directoryCustomerID = resp.Owner.DirectoryCustomerId
			return nil
		}
	}
	return nil
}
