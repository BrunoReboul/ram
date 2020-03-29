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

// Package getgroupsettings retreives one group settings from `Groups Settings API`
// - Triggered by: PubSub messages in GCI groups topic
// - Instances: Only one
// - Output: PubSub messages to a dedicated topic formated like Cloud Asset Inventory feed messages
// - Cardinality: one-one, one output message for each triggering event
package getgroupsettings

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/BrunoReboul/ram/ram"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/pubsub"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                   context.Context
	groupsSettingsService *groupssettings.Service
	initFailed            bool
	outputTopicName       string
	pubSubClient          *pubsub.Client
	retryTimeOutSeconds   int64
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var clientOption option.ClientOption
	var err error
	var gciAdminUserToImpersonate string
	var keyJSONFilePath string
	var ok bool
	var projectID string
	var serviceAccountEmail string

	gciAdminUserToImpersonate = os.Getenv("GCIADMINUSERTOIMPERSONATE")
	global.outputTopicName = os.Getenv("OUTPUTTOPICNAME")
	keyJSONFilePath = "./" + os.Getenv("KEYJSONFILENAME")
	projectID = os.Getenv("GCP_PROJECT")
	serviceAccountEmail = os.Getenv("SERVICEACCOUNTNAME")

	log.Println("Function COLD START")
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	if clientOption, ok = ram.GetClientOptionAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, projectID, gciAdminUserToImpersonate, []string{"https://www.googleapis.com/auth/apps.groups.settings"}); !ok {
		return
	}
	global.groupsSettingsService, err = groupssettings.NewService(ctx, clientOption)
	if err != nil {
		log.Printf("ERROR - groupssettings.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.pubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - pubsub.NewClient: %v", err)
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

	// Pass data to global variables to deal with func browseGroup
	var feedMessageGroup ram.FeedMessageGroup
	err = json.Unmarshal(PubSubMessage.Data, &feedMessageGroup)
	if err != nil {
		log.Println("ERROR - json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)")
		return nil // NO RETRY
	}

	var feedMessageGroupSettings ram.FeedMessageGroupSettings
	feedMessageGroupSettings.Window.StartTime = metadata.Timestamp
	feedMessageGroupSettings.Origin = feedMessageGroup.Origin
	feedMessageGroupSettings.Asset.Ancestors = feedMessageGroup.Asset.Ancestors
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Asset.Name = feedMessageGroup.Asset.Name + "/groupSettings"
	feedMessageGroupSettings.Deleted = feedMessageGroup.Deleted
	if !feedMessageGroup.Deleted {
		groupSettings, err := global.groupsSettingsService.Groups.Get(feedMessageGroup.Asset.Resource.Email).Do()
		if err != nil {
			return fmt.Errorf("groupsSettingsService.Groups.Get: %v", err) // RETRY
		}
		feedMessageGroupSettings.Asset.Resource = groupSettings
	}

	publishRequest := ram.PublishRequest{Topic: global.outputTopicName}
	feedMessageGroupSettingsJSON, err := json.Marshal(feedMessageGroupSettings)
	if err != nil {
		log.Println("ERROR - json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)")
		return nil // NO RETRY
	}
	pubSubMessage := &pubsub.Message{
		Data: feedMessageGroupSettingsJSON,
	}
	id, err := global.pubSubClient.Topic(publishRequest.Topic).Publish(global.ctx, pubSubMessage).Get(global.ctx)
	if err != nil {
		return fmt.Errorf("pubSubClient.Topic(publishRequest.Topic).Publish: %v", err) // RETRY
	}
	log.Printf("Group %s %s settings published to pubsub topic %s id %s %s", feedMessageGroup.Asset.Resource.Id, feedMessageGroup.Asset.Resource.Email, global.outputTopicName, id, string(feedMessageGroupSettingsJSON))

	return nil
}
