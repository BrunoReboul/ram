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

	"github.com/BrunoReboul/ram/utilities/aut"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/logging"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/BrunoReboul/ram/utilities/str"
	"github.com/google/uuid"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
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
	environment                 string
	firestoreClient             *firestore.Client
	GCIGroupMembersTopicName    string
	GCIGroupSettingsTopicName   string
	groupsSettingsService       *groupssettings.Service
	instanceName                string
	logEntry                    logEntry
	microserviceName            string
	organizationID              string
	projectID                   string
	PubSubID                    string
	pubsubPublisherClient       *pubsub.PublisherClient
	retriesNumber               time.Duration
	retryTimeOutSeconds         int64
	step                        logging.Step
	stepStack                   logging.Steps
	topicList                   []string
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var clientOption option.ClientOption
	var ok bool

	initID := fmt.Sprintf("%v", uuid.New())
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Println(logging.Entry{
			Severity:    "CRITICAL",
			Message:     "init_failed",
			Description: fmt.Sprintf("ReadUnmarshalYAML %s %v", solution.SettingsFileName, err),
			InitID:      initID,
		})
		return err
	}

	global.environment = instanceDeployment.Core.EnvironmentName
	global.instanceName = instanceDeployment.Core.InstanceName
	global.microserviceName = instanceDeployment.Core.ServiceName

	log.Println(logging.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

	gciAdminUserToImpersonate := instanceDeployment.Settings.Instance.GCI.SuperAdminEmail
	global.collectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.GCIGroupMembersTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	global.GCIGroupSettingsTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retriesNumber = instanceDeployment.Settings.Service.RetriesNumber
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := solution.PathToFunctionCode + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		instanceDeployment.Core.ServiceName,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)

	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("firestore.NewClient %v", err),
			InitID:           initID,
		})
		return err
	}

	serviceAccountKeyNames, err := gfs.ListKeyNames(ctx, global.firestoreClient, instanceDeployment.Core.ServiceName)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("gfs.ListKeyNames %v", err),
			InitID:           initID,
		})
		return err
	}

	if clientOption, ok = aut.GetClientOptionAndCleanKeys(ctx,
		serviceAccountEmail,
		keyJSONFilePath,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		gciAdminUserToImpersonate,
		[]string{"https://www.googleapis.com/auth/apps.groups.settings", "https://www.googleapis.com/auth/admin.directory.group.readonly"},
		serviceAccountKeyNames,
		initID,
		global.microserviceName,
		global.instanceName,
		global.environment); !ok {
		return fmt.Errorf("aut.GetClientOptionAndCleanKeys")
	}
	global.dirAdminService, err = admin.NewService(ctx, clientOption)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("admin.NewService %v", err),
			InitID:           initID,
		})
		return err
	}
	global.groupsSettingsService, err = groupssettings.NewService(ctx, clientOption)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("groupssettings.NewService %v", err),
			InitID:           initID,
		})
	}
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("global.pubsubPublisherClient %v", err),
			InitID:           initID,
		})
	}
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("cloudresourcemanager.NewService %v", err),
			InitID:           initID,
		})
	}
	err = gps.GetTopicList(global.ctx, global.pubsubPublisherClient, global.projectID, &global.topicList)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("gps.GetTopicList %v", err),
			InitID:           initID,
		})
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("pubsub_id no available metadata.FromContext: %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	global.PubSubID = metadata.EventID
	parts := strings.Split(metadata.Resource.Name, "/")
	global.step = logging.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.PubSubID),
		StepTimestamp: metadata.Timestamp,
	}
	global.stepStack = append(global.stepStack, global.step) // as the pubsub log entry is an initial step

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(logging.Entry{
		MicroserviceName:           global.microserviceName,
		InstanceName:               global.instanceName,
		Environment:                global.environment,
		Severity:                   "NOTICE",
		Message:                    "start",
		TriggeringPubsubID:         global.PubSubID,
		TriggeringPubsubAgeSeconds: d.Seconds(),
		TriggeringPubsubTimestamp:  &metadata.Timestamp,
		Now:                        &now,
	})

	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Println(logging.Entry{
			MicroserviceName:           global.microserviceName,
			InstanceName:               global.instanceName,
			Environment:                global.environment,
			Severity:                   "CRITICAL",
			Message:                    "noretry",
			Description:                "Pubsub message too old",
			TriggeringPubsubID:         global.PubSubID,
			TriggeringPubsubAgeSeconds: d.Seconds(),
			TriggeringPubsubTimestamp:  &metadata.Timestamp,
			Now:                        &now,
		})
		return nil
	}

	err = json.Unmarshal(PubSubMessage.Data, &global.logEntry)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal logentry %v %v", PubSubMessage.Data, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}

	switch global.logEntry.Resource.Type {
	case "audited_resource":
		switch global.logEntry.Resource.Labels["service"] {
		case "admin.googleapis.com":
			return convertAdminActivityEvent(global)
		default:
			log.Printf("pubsub_id %s NORETRY_ERROR unmanaged global.logEntry.Resource.Labels service  %s", global.PubSubID, global.logEntry.Resource.Labels["service"])
			now := time.Now()
			latency := now.Sub(global.step.StepTimestamp)
			latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
			log.Println(logging.Entry{
				MicroserviceName:     global.microserviceName,
				InstanceName:         global.instanceName,
				Environment:          global.environment,
				Severity:             "NOTICE",
				Message:              "cancel",
				Description:          fmt.Sprintf("unmanaged global.logEntry.Resource.Labels service  %s", global.logEntry.Resource.Labels["service"]),
				Now:                  &now,
				TriggeringPubsubID:   global.PubSubID,
				OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
				LatencySeconds:       latency.Seconds(),
				LatencyE2ESeconds:    latencyE2E.Seconds(),
				StepStack:            global.stepStack,
			})
			return nil
		}
	default:
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              "cancel",
			Description:          fmt.Sprintf("unmanaged logEntry.Resource.Type %s", global.logEntry.Resource.Type),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
		return nil
	}
}

// https://developers.google.com/admin-sdk/reports/v1/reference/activity-ref-appendix-a/admin-event-names
func convertAdminActivityEvent(global *Global) (err error) {
	var protoPayload protoPayload

	err = json.Unmarshal(global.logEntry.ProtoPayload, &protoPayload)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal protoPaylaod %v %v", global.logEntry.ProtoPayload, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}

	parts := strings.Split(protoPayload.ResourceName, "/")
	global.organizationID = parts[1]
	getCustomerID(global)
	if global.directoryCustomerID == "" {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "cannot get customer ID",
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}

	for _, event := range protoPayload.Metadata.Events {
		switch event.EventType {
		case "GROUP_SETTINGS":
			return convertGroupSettings(&event, global)
		default:
			now := time.Now()
			latency := now.Sub(global.step.StepTimestamp)
			latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
			log.Println(logging.Entry{
				MicroserviceName:     global.microserviceName,
				InstanceName:         global.instanceName,
				Environment:          global.environment,
				Severity:             "NOTICE",
				Message:              "cancel",
				Description:          fmt.Sprintf("unmanaged event.EventType %s", event.EventType),
				Now:                  &now,
				TriggeringPubsubID:   global.PubSubID,
				OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
				LatencySeconds:       latency.Seconds(),
				LatencyE2ESeconds:    latencyE2E.Seconds(),
				StepStack:            global.stepStack,
			})
			return nil
		}
	}
	return nil
}
func getCustomerID(global *Global) {
	documentID := fmt.Sprintf("//cloudresourcemanager.googleapis.com/organizations/%s", global.organizationID)
	documentID = str.RevertSlash(documentID)
	documentPath := global.collectionID + "/" + documentID
	// log.Printf("documentPath %s", documentPath)
	documentSnap, found := gfs.GetDoc(global.ctx, global.firestoreClient, documentPath, global.retriesNumber)
	if found {
		// log.Printf("Found firestore document %s", documentPath)

		assetMap := documentSnap.Data()
		assetMapJSON, err := json.Marshal(assetMap)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            "json.Marshal(assetMap)",
				Description:        fmt.Sprintf("assetMap %v err %v", assetMap, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return
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
							return
						}
					}
				}
			}
		}
	} else {
		log.Printf("pubsub_id %s WARNING not found in firestore %s", global.PubSubID, documentPath)
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "WARNING",
			Message:            "not found in firestore",
			Description:        fmt.Sprintf("documentPath %s", documentPath),
			TriggeringPubsubID: global.PubSubID,
		})
		//try resourcemamager API
		resp, err := global.cloudresourcemanagerService.Organizations.Get(global.organizationID).Context(global.ctx).Do()
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            "not found with resource manager",
				Description:        fmt.Sprintf("cloudresourcemanagerService.Organizations.Get orgID %s err %v ", global.organizationID, err),
				TriggeringPubsubID: global.PubSubID,
			})
		} else {
			global.directoryCustomerID = resp.Owner.DirectoryCustomerId
			return
		}
	}
	return
}

func convertGroupSettings(event *event, global *Global) (err error) {
	var parameters groupSettingsParameters
	err = json.Unmarshal(event.Parameter, &parameters)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal(event.Parameter, &parameters) %v %v", event.Parameter, err),
			TriggeringPubsubID: global.PubSubID,
		})
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
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("expected parameter GROUP_EMAIL not found, insertId %s", global.logEntry.InsertID),
			TriggeringPubsubID: global.PubSubID,
		})
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
			log.Printf("pubsub_id %s NORETRY_ERROR ADD_GROUP_MEMBER expected parameter USER_EMAIL aka member, not found, insertId %s", global.PubSubID, global.logEntry.InsertID)
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
			log.Printf("pubsub_id %s NORETRY_ERROR ADD_GROUP_MEMBER expected parameter USER_EMAIL aka member, not found, insertId %s", global.PubSubID, global.logEntry.InsertID)
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "noretry",
				Description:        fmt.Sprintf("ADD_GROUP_MEMBER expected parameter USER_EMAIL aka member, not found, insertId %s", global.logEntry.InsertID),
				TriggeringPubsubID: global.PubSubID,
			})
			return nil
		}
		return publishGroupMemberDeletion(groupEmail, memberEmail, global)
	case "CHANGE_GROUP_SETTING":
		// https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-group-settings#CHANGE_GROUP_SETTING
		return publishGroupSettings(groupEmail, global)
	default:
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              "cancel",
			Description:          fmt.Sprintf("unmanaged event.EventName %s", event.EventName),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
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
	feedMessage.StepStack = global.stepStack
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
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "INFO",
				Message:            "cleaning cache group orphans",
				Description:        fmt.Sprintf("iteration %d", i),
				TriggeringPubsubID: global.PubSubID,
			})
		}
		i++
		documentSnap, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("publishGroupDeletion iter.Next() %v", err)
		}
		if documentSnap.Exists() {
			found = true
			err = documentSnap.DataTo(&retreivedFeedMessageGroup)
			if err != nil {
				return fmt.Errorf("publishGroupDeletion documentSnap.DataTo %v", err)
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
				return fmt.Errorf("publishGroup(retreivedFeedMessageGroup %v", err)
			}
		} else {
			return fmt.Errorf("document does not exist %s", documentSnap.Ref.Path)
		}
	}
	if !found {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "WARNING",
			Message:            "deleted group not found in cache, cannot clean up RAM data",
			Description:        fmt.Sprintf("groupEmail %s", groupEmail),
			TriggeringPubsubID: global.PubSubID,
		})
	}
	return nil
}

func publishGroup(feedMessage interface{}, isDeleted bool, groupEmail string, assetName string, global *Global) (err error) {
	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("publishGroup json.Marshal(feedMessage) %v %v", feedMessage, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	topicShortName := fmt.Sprintf("gci-groups-%s", global.directoryCustomerID)
	if err = gps.CreateTopic(global.ctx, global.pubsubPublisherClient, &global.topicList, topicShortName, global.projectID); err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("gps.CreateTopic %s %v", topicShortName, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	topicName := fmt.Sprintf("projects/%s/topics/%s", global.projectID, topicShortName)
	publishRequest.Topic = topicName
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		// log.Printf("publish err no nil %v", err)
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", topicShortName, err)
	}
	now := time.Now()
	latency := now.Sub(global.step.StepTimestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(logging.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              "finish",
		Description:          fmt.Sprintf("group published to pubsub %s (isdeleted status=%v) %s topic %s ids %v %s", groupEmail, isDeleted, assetName, topicName, pubsubResponse.MessageIds, string(feedMessageJSON)),
		Now:                  &now,
		TriggeringPubsubID:   global.PubSubID,
		OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
	})
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
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "INFO",
				Message:            "cleaning cache groupMember orphans",
				Description:        fmt.Sprintf("iteration %d", i),
				TriggeringPubsubID: global.PubSubID,
			})
		}
		i++
		documentSnap, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("publishGroupMemberDeletion iter.Next() %v", err)
		}
		if documentSnap.Exists() {
			found = true
			err = documentSnap.DataTo(&retreivedFeedMessageGroupMember)
			if err != nil {
				return fmt.Errorf("publishGroupMemberDeletion documentSnap.DataTo %v", err)
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
				return fmt.Errorf("publishGroup(retreivedFeedMessageGroupMember %v", err)
			}
		} else {
			return fmt.Errorf("document does not exist %s", documentSnap.Ref.Path)
		}
	}
	if !found {
		log.Printf("pubsub_id %s NORETRY_ERROR deleted groupMember not found in cache, cannot clean up RAM data member %s group %s", global.PubSubID, memberEmail, groupEmail)
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "WARNING",
			Message:            "deleted groupMember not found in cache, cannot clean up RAM data",
			Description:        fmt.Sprintf("member %s group %s", memberEmail, groupEmail),
			TriggeringPubsubID: global.PubSubID,
		})
	}
	return nil
}

func publishGroupMember(feedMessage interface{}, isDeleted bool, groupEmail string, memberEmail string, assetName string, global *Global) (err error) {
	feedMessageJSON, err := json.Marshal(feedMessage)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR %s json.Marshal(feedMessage): %v", global.PubSubID, assetName, err)
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("publishGroupMember json.Marshal(feedMessage) %v %v", feedMessage, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
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
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", publishRequest.Topic, err)
	}
	now := time.Now()
	latency := now.Sub(global.step.StepTimestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(logging.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              "finish",
		Description:          fmt.Sprintf("groupMember published to pubsub %s in group %s (isdeleted status=%v) %s topic %s ids %v %s", memberEmail, groupEmail, isDeleted, assetName, global.GCIGroupMembersTopicName, pubsubResponse.MessageIds, string(feedMessageJSON)),
		Now:                  &now,
		TriggeringPubsubID:   global.PubSubID,
		OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
	})
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
		return fmt.Errorf("groupsSettingsService.Groups.Get: %s %v", groupEmail, err)
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
		log.Printf("pubsub_id %s NORETRY_ERROR json.Marshal(feedMessageGroupSettings)", global.PubSubID)
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Marshal(feedMessageGroupSettings) %v %v", feedMessageGroupSettings, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
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
		return fmt.Errorf("%s global.pubsubPublisherClient.Publish: %v", publishRequest.Topic, err)
	}
	now := time.Now()
	latency := now.Sub(global.step.StepTimestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(logging.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              "finish",
		Description:          fmt.Sprintf("groupSettings published to pubsub %s (isdeleted status=%v) %s topic %s ids %v %s", feedMessageGroupSettings.Asset.Resource.Email, feedMessageGroupSettings.Deleted, feedMessageGroupSettings.Asset.Name, global.GCIGroupSettingsTopicName, pubsubResponse.MessageIds, string(feedMessageGroupSettingsJSON)),
		Now:                  &now,
		TriggeringPubsubID:   global.PubSubID,
		OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
	})
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
