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
	"time"

	"github.com/BrunoReboul/ram/utilities/aut"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                   context.Context
	firestoreClient       *firestore.Client
	groupsSettingsService *groupssettings.Service
	outputTopicName       string
	projectID             string
	PubSubID              string
	pubsubPublisherClient *pubsub.PublisherClient
	retryTimeOutSeconds   int64
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var clientOption option.ClientOption
	var ok bool

	log.Println("Function COLD START")
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("ReadUnmarshalYAML %s %v", solution.SettingsFileName, err)
	}

	gciAdminUserToImpersonate := instanceDeployment.Settings.Instance.GCI.SuperAdminEmail
	global.outputTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := solution.PathToFunctionCode + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		instanceDeployment.Core.ServiceName,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)

	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
	if err != nil {
		return fmt.Errorf("firestore.NewClient: %v", err)
	}

	serviceAccountKeyNames, err := gfs.ListKeyNames(ctx, global.firestoreClient, instanceDeployment.Core.ServiceName)
	if err != nil {
		return fmt.Errorf("gfs.ListKeyNames %v", err)
	}

	if clientOption, ok = aut.GetClientOptionAndCleanKeys(ctx,
		serviceAccountEmail,
		keyJSONFilePath,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		gciAdminUserToImpersonate,
		[]string{"https://www.googleapis.com/auth/apps.groups.settings", "https://www.googleapis.com/auth/admin.directory.group.readonly"},
		serviceAccountKeyNames); !ok {
		return fmt.Errorf("aut.GetClientOptionAndCleanKeys")
	}
	global.groupsSettingsService, err = groupssettings.NewService(ctx, clientOption)
	if err != nil {
		return fmt.Errorf("groupssettings.NewService: %v", err)
	}
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		return fmt.Errorf("global.pubsubPublisherClient: %v", err)
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return fmt.Errorf("pubsub_id no available REDO_ON_TRANSIENT metadata.FromContext: %v", err)
	}
	global.PubSubID = metadata.EventID
	expiration := metadata.Timestamp.Add(time.Duration(global.retryTimeOutSeconds) * time.Second)
	if time.Now().After(expiration) {
		log.Printf("pubsub_id %s NORETRY_ERROR pubsub message too old", global.PubSubID)
		return nil
	}

	// Pass data to global variables to deal with func browseGroup
	var feedMessageGroup cai.FeedMessageGroup
	err = json.Unmarshal(PubSubMessage.Data, &feedMessageGroup)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)", global.PubSubID)
		return nil
	}

	var feedMessageGroupSettings cai.FeedMessageGroupSettings
	feedMessageGroupSettings.Window.StartTime = metadata.Timestamp
	feedMessageGroupSettings.Origin = feedMessageGroup.Origin
	feedMessageGroupSettings.Asset.Ancestors = feedMessageGroup.Asset.Ancestors
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Asset.Name = feedMessageGroup.Asset.Name + "/groupSettings"
	feedMessageGroupSettings.Deleted = feedMessageGroup.Deleted
	if !feedMessageGroup.Deleted {
		groupSettings, err := global.groupsSettingsService.Groups.Get(feedMessageGroup.Asset.Resource.Email).Do()
		if err != nil {
			return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT groupsSettingsService.Groups.Get: %v", global.PubSubID, err)
		}
		feedMessageGroupSettings.Asset.Resource = groupSettings
	}

	feedMessageGroupSettingsJSON, err := json.Marshal(feedMessageGroupSettings)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)", global.PubSubID)
		return nil
	}

	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = feedMessageGroupSettingsJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, global.outputTopicName)
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT global.pubsubPublisherClient.Publish: %v", global.PubSubID, err)
	}
	log.Printf("pubsub_id %s  group settings %s isdeleted: %v %s published to pubsub topic %s ids %v %s",
		global.PubSubID,
		feedMessageGroup.Asset.Resource.Email,
		feedMessageGroup.Deleted,
		feedMessageGroup.Asset.Resource.Id,
		global.outputTopicName,
		pubsubResponse.MessageIds,
		string(feedMessageGroupSettingsJSON))

	return nil
}
