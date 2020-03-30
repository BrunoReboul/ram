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

package getgroupsettings

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	// pubsubold "cloud.google.com/go/pubsub"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                   context.Context
	groupsSettingsService *groupssettings.Service
	initFailed            bool
	outputTopicName       string
	// pubSubClient          *pubsubold.Client
	pubsubPublisherClient *pubsub.PublisherClient
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
	// global.pubSubClient, err = pubsubold.NewClient(ctx, projectID)
	// if err != nil {
	// 	log.Printf("ERROR - pubsubold.NewClient: %v", err)
	// 	global.initFailed = true
	// 	return
	// }
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		log.Printf("ERROR - global.pubsubPublisherClient: %v", err)
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

	feedMessageGroupSettingsJSON, err := json.Marshal(feedMessageGroupSettings)
	if err != nil {
		log.Println("ERROR - json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)")
		return nil // NO RETRY
	}

	// publishRequest := ram.PublishRequest{Topic: global.outputTopicName}
	// pubSubMessage := &pubsubold.Message{
	// 	Data: feedMessageGroupSettingsJSON,
	// }
	// id, err := global.pubSubClient.Topic(publishRequest.Topic).Publish(global.ctx, pubSubMessage).Get(global.ctx)
	// if err != nil {
	// 	return fmt.Errorf("pubSubClient.Topic(publishRequest.Topic).Publish: %v", err) // RETRY
	// }
	// log.Printf("Group %s %s settings published to pubsub topic %s id %s %s", feedMessageGroup.Asset.Resource.Id, feedMessageGroup.Asset.Resource.Email, global.outputTopicName, id, string(feedMessageGroupSettingsJSON))

	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageGroupSettingsJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequestv1 pubsubpb.PublishRequest
	publishRequestv1.Topic = global.outputTopicName
	publishRequestv1.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequestv1)
	if err != nil {
		return fmt.Errorf("global.pubsubPublisherClient.Publish: %v", err) // RETRY
	}
	log.Printf("Group %s %s settings published to pubsub topic %s ids %v %s", feedMessageGroup.Asset.Resource.Id, feedMessageGroup.Asset.Resource.Email, global.outputTopicName, pubsubResponse.MessageIds, string(feedMessageGroupSettingsJSON))

	return nil
}
