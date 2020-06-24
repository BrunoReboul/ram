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
	"time"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	admin "google.golang.org/api/admin/directory/v1"
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
	Metadata struct {
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
	ctx                       context.Context
	dirAdminService           *admin.Service
	groupsSettingsService     *groupssettings.Service
	firestoreClient           *firestore.Client
	initFailed                bool
	GCIGroupMembersTopicName  string
	GCIGroupSettingsTopicName string
	projectID                 string
	pubsubPublisherClient     *pubsub.PublisherClient
	retryTimeOutSeconds       int64
	logEntry                  logEntry
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
	global.GCIGroupMembersTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	global.GCIGroupSettingsTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := "./" + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := os.Getenv("FUNCTION_IDENTITY")

	if clientOption, ok = ram.GetClientOptionAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, global.projectID, gciAdminUserToImpersonate, []string{"https://www.googleapis.com/auth/apps.groups.settings"}); !ok {
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
	feedMessageGroupSettings.Origin = "real-time"
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Deleted = isDeleted

	if !isDeleted {
		groupSettings, err := global.groupsSettingsService.Groups.Get(groupEmail).Do()
		if err != nil {
			return fmt.Errorf("groupsSettingsService.Groups.Get: %v", err) // RETRY
		}
		feedMessageGroupSettings.Asset.Resource = groupSettings
	}

	// feedMessageGroupSettings.Asset.Ancestors = feedMessageGroup.Asset.Ancestors
	// feedMessageGroupSettings.Asset.Name = feedMessageGroup.Asset.Name + "/groupSettings"

	log.Printf("CHANGE_GROUP_SETTING %s %v", groupEmail, feedMessageGroupSettings.Asset.Resource)

	return nil
}
