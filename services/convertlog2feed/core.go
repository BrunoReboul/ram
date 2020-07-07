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
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/iterator"
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
	cloudresourcemanagerService *cloudresourcemanager.Service
	collectionID                string
	ctx                         context.Context
	dirAdminService             *admin.Service
	directoryCustomerID         string
	firestoreClient             *firestore.Client
	GCIGroupMembersTopicName    string
	GCIGroupSettingsTopicName   string
	groupsSettingsService       *groupssettings.Service
	initFailed                  bool
	logEntry                    logEntry
	organizationID              string
	projectID                   string
	pubsubPublisherClient       *pubsub.PublisherClient
	retriesNumber               time.Duration
	retryTimeOutSeconds         int64
	topicList                   []string
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
	err = ram.ReadUnmarshalYAML(ram.PathToFunctionCode+ram.SettingsFileName, &instanceDeployment)
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
	global.retriesNumber = instanceDeployment.Settings.Service.RetriesNumber
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := ram.PathToFunctionCode + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		instanceDeployment.Core.ServiceName,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)

	if clientOption, ok = ram.GetClientOptionAndCleanKeys(ctx,
		serviceAccountEmail,
		keyJSONFilePath,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		gciAdminUserToImpersonate,
		[]string{"https://www.googleapis.com/auth/apps.groups.settings", "https://www.googleapis.com/auth/admin.directory.group.readonly"}); !ok {
		global.initFailed = true
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
	err = gps.GetTopicList(global.ctx, global.pubsubPublisherClient, global.projectID, &global.topicList)
	if err != nil {
		log.Printf("ERROR - gps.GetTopicList: %v", err)
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
	if global.directoryCustomerID == "" {
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
func getCustomerID(global *Global) (err error) {
	documentID := fmt.Sprintf("//cloudresourcemanager.googleapis.com/organizations/%s", global.organizationID)
	documentID = ram.RevertSlash(documentID)
	documentPath := global.collectionID + "/" + documentID
	// log.Printf("documentPath %s", documentPath)
	documentSnap, found := ram.FireStoreGetDoc(global.ctx, global.firestoreClient, documentPath, global.retriesNumber)
	if found {
		// log.Printf("Found firestore document %s", documentPath)

		assetMap := documentSnap.Data()
		assetMapJSON, err := json.Marshal(assetMap)
		if err != nil {
			log.Println("ERROR - json.Marshal(assetMap)")
			return nil // NO RETRY
		}
		// log.Printf("%s", string(assetMapJSON))
		_ = assetMapJSON

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

func convertGroupSettings(event *event, global *Global) (err error) {
	var parameters groupSettingsParameters
	err = json.Unmarshal(event.Parameter, &parameters)
	if err != nil {
		log.Printf("ERROR json.Unmarshal groupSettingsParameters %v", err)
		return nil
	}
	var groupEmail string
	for _, parameter := range parameters {
		switch parameter.Name {
		case "GROUP_EMAIL":
			groupEmail = strings.ToLower(parameter.Value)
			// log.Printf("groupEmail %s", groupEmail)
		}
	}
	if groupEmail == "" {
		log.Printf("ERROR expected parameter GROUP_EMAIL not found, insertId %s", global.logEntry.InsertID)
		return nil
	}
	switch event.EventName {
	case "CREATE_GROUP", "CHANGE_GROUP_NAME", "CHANGE_GROUP_DESCRIPTION":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#CREATE_GROUP
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#CHANGE_GROUP_NAME
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#CHANGE_GROUP_DESCRIPTION
		return publishGroupCreationOrUpdate(groupEmail, global)
	case "DELETE_GROUP":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#DELETE_GROUP
		return publishGroupDeletion(groupEmail, global)
	case "ADD_GROUP_MEMBER", "UPDATE_GROUP_MEMBER":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#ADD_GROUP_MEMBER
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#UPDATE_GROUP_MEMBER
		var memberEmail string
		for _, parameter := range parameters {
			switch parameter.Name {
			case "USER_EMAIL":
				// The parmeter is no only a user email. It is a member email, can be group, service account or user
				memberEmail = parameter.Value
			}
		}
		if memberEmail == "" {
			log.Printf("ERROR ADD_GROUP_MEMBER expected parameter USER_EMAIL aka member, not found, insertId %s", global.logEntry.InsertID)
			return nil
		}
		return publishGroupMemberCreationOrUpdate(groupEmail, memberEmail, global)
	case "REMOVE_GROUP_MEMBER":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#REMOVE_GROUP_MEMBER
		var memberEmail string
		for _, parameter := range parameters {
			switch parameter.Name {
			case "USER_EMAIL":
				// The parmeter is no only a user email. It is a member email, can be group, service account or user
				memberEmail = parameter.Value
			}
		}
		if memberEmail == "" {
			log.Printf("ERROR ADD_GROUP_MEMBER expected parameter USER_EMAIL aka member, not found, insertId %s", global.logEntry.InsertID)
			return nil
		}
		return publishGroupMemberDeletion(groupEmail, memberEmail, global)
	case "CHANGE_GROUP_SETTING":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#CHANGE_GROUP_SETTING
		return publishGroupSettings(groupEmail, global)
	default:
		log.Printf("Unmanaged event.EventName %s", event.EventName)
		return nil
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#GROUP_LIST_DOWNLOAD
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#UPDATE_GROUP_MEMBER_DELIVERY_SETTINGS
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#UPDATE_GROUP_MEMBER_DELIVERY_SETTINGS_CAN_EMAIL_OVERRIDE
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#GROUP_MEMBER_BULK_UPLOAD
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#GROUP_MEMBERS_DOWNLOAD
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#WHITELISTED_GROUPS_UPDATED
	}
}

func publishGroupCreationOrUpdate(groupEmail string, global *Global) (err error) {
	group, err := getGroupFromEmail(groupEmail, global)
	if err != nil {
		return err
	}
	var feedMessage cai.FeedMessageGroup
	feedMessage.Window.StartTime = global.logEntry.Timestamp
	feedMessage.Origin = "real-time-log-export"
	feedMessage.Deleted = false
	feedMessage.Asset.Ancestors = []string{fmt.Sprintf("directories/%s", global.directoryCustomerID)}
	feedMessage.Asset.AncestryPath = fmt.Sprintf("directories/%s", global.directoryCustomerID)
	feedMessage.Asset.AssetType = "www.googleapis.com/admin/directory/groups"
	feedMessage.Asset.Name = fmt.Sprintf("//directories/%s/groups/%s", global.directoryCustomerID, group.Id)
	feedMessage.Asset.Resource = group
	feedMessage.Asset.Resource.Etag = ""
	return publishGroup(feedMessage, feedMessage.Deleted, groupEmail, feedMessage.Asset.Name, global)
}

func publishGroupDeletion(groupEmail string, global *Global) (err error) {
	assets := global.firestoreClient.Collection(global.collectionID)
	query := assets.Where(
		"asset.assetType", "==", "www.googleapis.com/admin/directory/groups").Where(
		"asset.resource.email", "==", strings.ToLower(groupEmail))
	var documentSnap *firestore.DocumentSnapshot
	iter := query.Documents(global.ctx)
	defer iter.Stop()
	// multiple documents may be found in case of orphans in cache
	type cachedFeedMessageGroup struct {
		Asset struct {
			Name         string   `firestore:"name" json:"name"`
			AssetType    string   `firestore:"assetType" json:"assetType"`
			Ancestors    []string `firestore:"ancestors" json:"ancestors"`
			AncestryPath string   `firestore:"ancestryPath" json:"ancestryPath"`
			Resource     struct {
				Email string `firestore:"email" json:"email"`
				ID    string `firestore:"id" json:"id"`
				Kind  string `firestore:"kind" json:"kind"`
				Name  string `firestore:"name" json:"name"`
			} `firestore:"resource" json:"resource"`
		} `firestore:"asset" json:"asset"`
		Deleted bool   `firestore:"deleted" json:"deleted"`
		Origin  string `firestore:"origin" json:"origin"`
		Window  struct {
			StartTime time.Time `firestore:"startTime" json:"startTime"`
		} `firestore:"window" json:"window"`
	}
	var retreivedFeedMessageGroup cachedFeedMessageGroup
	found := false
	var i int64
	for {
		if i > 0 {
			log.Printf("Cleaning cache group orphans iteration %d", i)
		}
		i++
		documentSnap, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("publishGroupDeletion iter.Next() %v", err) // RETRY
		}
		if documentSnap.Exists() {
			found = true
			err = documentSnap.DataTo(&retreivedFeedMessageGroup)
			if err != nil {
				return fmt.Errorf("publishGroupDeletion documentSnap.DataTo %v", err) // RETRY
			}

			// Updating fields
			retreivedFeedMessageGroup.Window.StartTime = global.logEntry.Timestamp
			retreivedFeedMessageGroup.Origin = "real-time-log-export"
			retreivedFeedMessageGroup.Deleted = true

			err = publishGroup(retreivedFeedMessageGroup,
				retreivedFeedMessageGroup.Deleted,
				retreivedFeedMessageGroup.Asset.Resource.Email,
				retreivedFeedMessageGroup.Asset.Name,
				global)
			if err != nil {
				return fmt.Errorf("publishGroup(retreivedFeedMessageGroup %v", err) // RETRY
			}
		} else {
			return fmt.Errorf("document does not exist %s", documentSnap.Ref.Path) // RETRY
		}
	}
	if !found {
		log.Printf("ERROR - deleted group not found in cache, cannot clean up RAM data %s", groupEmail)
	}
	return nil
}

func publishGroup(feedMessage interface{}, isDeleted bool, groupEmail string, assetName string, global *Global) (err error) {
	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Printf("ERROR - %s json.Marshal(feedMessage): %v", assetName, err)
		return nil // NO RETRY
	}
	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	topicShortName := fmt.Sprintf("gci-groups-%s", global.directoryCustomerID)
	if err = gps.CreateTopic(global.ctx, global.pubsubPublisherClient, &global.topicList, topicShortName, global.projectID); err != nil {
		log.Printf("ERROR - %s gps.CreateTopic: %v", topicShortName, err)
		return nil // NO RETRY
	}
	topicName := fmt.Sprintf("projects/%s/topics/%s", global.projectID, topicShortName)
	publishRequest.Topic = topicName
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		log.Printf("publish err no nil %v", err)
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", topicShortName, err) // RETRY
	}

	log.Printf("PubOps group %s isdeleted: %v %s published to pubsub topic %s ids %v %s",
		groupEmail,
		isDeleted,
		assetName,
		topicName,
		pubsubResponse.MessageIds,
		string(feedMessageJSON))
	return nil
}

func publishGroupMemberCreationOrUpdate(groupEmail string, memberEmail string, global *Global) (err error) {
	var groupMember *admin.Member
	var groupID string
	// groupKey: The value can be the group's email address, group alias, or the unique group ID.
	// memberKey: The value can be the member's (group or user) primary email address, alias, or unique ID
	// https://developers.google.com/admin-sdk/directory/v1/reference/members/get
	groupMember, err = global.dirAdminService.Members.Get(groupEmail, memberEmail).Context(global.ctx).Do()
	if err != nil {
		return fmt.Errorf("dirAdminService.Members.Get %v", err)
	}
	group, err := getGroupFromEmail(groupEmail, global)
	if err != nil {
		return err
	}
	groupID = group.Id

	var feedMessage cai.FeedMessageMember
	feedMessage.Asset.Ancestors = []string{
		fmt.Sprintf("groups/%s", groupID),
		fmt.Sprintf("directories/%s", global.directoryCustomerID)}

	feedMessage.Asset.AncestryPath = fmt.Sprintf("directories/%s/groups/%s", global.directoryCustomerID, groupID)
	feedMessage.Asset.Name = "//" + feedMessage.Asset.AncestryPath + "/members/" + groupMember.Id
	feedMessage.Asset.AssetType = "www.googleapis.com/admin/directory/members"
	feedMessage.Asset.Resource.GroupEmail = groupEmail
	feedMessage.Asset.Resource.MemberEmail = memberEmail
	feedMessage.Asset.Resource.ID = groupMember.Id
	feedMessage.Asset.Resource.Kind = groupMember.Kind
	feedMessage.Asset.Resource.Role = groupMember.Role
	feedMessage.Asset.Resource.Type = groupMember.Type

	feedMessage.Window.StartTime = global.logEntry.Timestamp
	feedMessage.Origin = "real-time-log-export"
	feedMessage.Deleted = false
	return publishGroupMember(feedMessage, feedMessage.Deleted, groupEmail, memberEmail, feedMessage.Asset.Name, global)
}

func publishGroupMemberDeletion(groupEmail string, memberEmail string, global *Global) (err error) {
	assets := global.firestoreClient.Collection(global.collectionID)
	query := assets.Where(
		"asset.assetType", "==", "www.googleapis.com/admin/directory/members").Where(
		"asset.resource.groupEmail", "==", strings.ToLower(groupEmail)).Where(
		"asset.resource.memberEmail", "==", strings.ToLower(memberEmail))
	var documentSnap *firestore.DocumentSnapshot
	iter := query.Documents(global.ctx)
	defer iter.Stop()
	// multiple documents may be found in case of orphans in cache
	type cachedFeedMessageGroupMember struct {
		Asset struct {
			Name         string   `firestore:"name" json:"name"`
			AssetType    string   `firestore:"assetType" json:"assetType"`
			Ancestors    []string `firestore:"ancestors" json:"ancestors"`
			AncestryPath string   `firestore:"ancestryPath" json:"ancestryPath"`
			Resource     struct {
				GroupEmail  string `firestore:"groupEmail" json:"groupEmail"`
				ID          string `firestore:"id" json:"id"`
				Kind        string `firestore:"kind" json:"kind"`
				MemberEmail string `firestore:"memberEmail" json:"memberEmail"`
				Role        string `firestore:"role" json:"role"`
				Type        string `firestore:"type" json:"type"`
			} `firestore:"resource" json:"resource"`
		} `firestore:"asset" json:"asset"`
		Deleted bool   `firestore:"deleted" json:"deleted"`
		Origin  string `firestore:"origin" json:"origin"`
		Window  struct {
			StartTime time.Time `firestore:"startTime" json:"startTime"`
		} `firestore:"window" json:"window"`
	}
	var retreivedFeedMessageGroupMember cachedFeedMessageGroupMember
	found := false
	var i int64
	for {
		if i > 0 {
			log.Printf("Cleaning cache groupMember orphans iteration %d", i)
		}
		i++
		documentSnap, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("publishGroupMemberDeletion iter.Next() %v", err) // RETRY
		}
		if documentSnap.Exists() {
			found = true
			err = documentSnap.DataTo(&retreivedFeedMessageGroupMember)
			if err != nil {
				return fmt.Errorf("publishGroupMemberDeletion documentSnap.DataTo %v", err) // RETRY
			}

			// Updating fields
			retreivedFeedMessageGroupMember.Window.StartTime = global.logEntry.Timestamp
			retreivedFeedMessageGroupMember.Origin = "real-time-log-export"
			retreivedFeedMessageGroupMember.Deleted = true

			err = publishGroupMember(retreivedFeedMessageGroupMember,
				retreivedFeedMessageGroupMember.Deleted,
				retreivedFeedMessageGroupMember.Asset.Resource.GroupEmail,
				retreivedFeedMessageGroupMember.Asset.Resource.MemberEmail,
				retreivedFeedMessageGroupMember.Asset.Name,
				global)
			if err != nil {
				return fmt.Errorf("publishGroup(retreivedFeedMessageGroupMember %v", err) // RETRY
			}
		} else {
			return fmt.Errorf("document does not exist %s", documentSnap.Ref.Path) // RETRY
		}
	}
	if !found {
		log.Printf("ERROR - deleted groupMember not found in cache, cannot clean up RAM data member %s group %s", memberEmail, groupEmail)
	}
	return nil
}

func publishGroupMember(feedMessage interface{}, isDeleted bool, groupEmail string, memberEmail string, assetName string, global *Global) (err error) {
	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Printf("ERROR - %s json.Marshal(feedMessage): %v", assetName, err)
		return nil // NO RETRY
	}
	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, global.GCIGroupMembersTopicName)
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", publishRequest.Topic, err) // RETRY
	}
	log.Printf("PubOps groupMember %s group %s isdeleted: %v %s published to pubsub topic %s ids %v %s",
		memberEmail,
		groupEmail,
		isDeleted,
		assetName,
		global.GCIGroupMembersTopicName,
		pubsubResponse.MessageIds,
		string(feedMessageJSON))

	return nil
}

func publishGroupSettings(groupEmail string, global *Global) (err error) {
	var feedMessageGroupSettings cai.FeedMessageGroupSettings
	feedMessageGroupSettings.Window.StartTime = global.logEntry.Timestamp
	feedMessageGroupSettings.Origin = "real-time-log-export"
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Deleted = false

	var groupID string
	groupSettings, err := global.groupsSettingsService.Groups.Get(groupEmail).Do()
	if err != nil {
		return fmt.Errorf("groupsSettingsService.Groups.Get: %s %v", groupEmail, err) // RETRY
	}
	feedMessageGroupSettings.Asset.Resource = groupSettings

	group, err := getGroupFromEmail(groupEmail, global)
	if err != nil {
		return err
	}
	groupID = group.Id

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
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", publishRequest.Topic, err) // RETRY
	}
	log.Printf("PubOps groupSettings %s isdeleted: %v %s published to pubsub topic %s ids %v %s",
		feedMessageGroupSettings.Asset.Resource.Email,
		feedMessageGroupSettings.Deleted,
		feedMessageGroupSettings.Asset.Name,
		global.GCIGroupSettingsTopicName,
		pubsubResponse.MessageIds,
		string(feedMessageGroupSettingsJSON))
	return nil
}

func getGroupFromEmail(groupEmail string, global *Global) (group *admin.Group, err error) {
	// groupKey: The value can be the group's email address, group alias, or the unique group ID.
	// https://developers.google.com/admin-sdk/directory/v1/reference/groups/get
	group, err = global.dirAdminService.Groups.Get(groupEmail).Context(global.ctx).Do()
	if err != nil {
		return group, fmt.Errorf("dirAdminService.Groups.Get %v", err) //
	}
	return group, nil
}
