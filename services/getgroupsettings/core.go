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
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/aut"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/glo"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

const waitSecOnQuotaExceeded = 70

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                   context.Context
	environment           string
	firestoreClient       *firestore.Client
	groupsSettingsService *groupssettings.Service
	instanceName          string
	microserviceName      string
	outputTopicName       string
	projectID             string
	PubSubID              string
	pubsubPublisherClient *pubsub.PublisherClient
	retryTimeOutSeconds   int64
	step                  glo.Step
	stepStack             glo.Steps
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
		log.Println(glo.Entry{
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

	log.Println(glo.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

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
		log.Println(glo.Entry{
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
		log.Println(glo.Entry{
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
		[]string{"https://www.googleapis.com/auth/apps.groups.settings"},
		serviceAccountKeyNames,
		initID,
		global.microserviceName,
		global.instanceName,
		global.environment); !ok {
		return fmt.Errorf("aut.GetClientOptionAndCleanKeys")
	}
	global.groupsSettingsService, err = groupssettings.NewService(ctx, clientOption)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("groupssettings.NewService %v", err),
			InitID:           initID,
		})
		return err
	}
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("global.pubsubPublisherClient %v", err),
			InitID:           initID,
		})
		return err
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(glo.Entry{
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
	global.stepStack = nil
	global.PubSubID = metadata.EventID
	parts := strings.Split(metadata.Resource.Name, "/")
	global.step = glo.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.PubSubID),
		StepTimestamp: metadata.Timestamp,
	}

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(glo.Entry{
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
		log.Println(glo.Entry{
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

	// Pass data to global variables to deal with func browseGroup
	var feedMessageGroup cai.FeedMessageGroup
	err = json.Unmarshal(PubSubMessage.Data, &feedMessageGroup)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)", global.PubSubID)
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal(pubSubMessage.Data, &feedMessageGroup) %v %v", PubSubMessage.Data, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	if feedMessageGroup.StepStack != nil {
		global.stepStack = append(feedMessageGroup.StepStack, global.step)
	} else {
		global.stepStack = append(global.stepStack, global.step)
	}

	var feedMessageGroupSettings cai.FeedMessageGroupSettings
	feedMessageGroupSettings.Window.StartTime = metadata.Timestamp
	feedMessageGroupSettings.Origin = feedMessageGroup.Origin
	feedMessageGroupSettings.Asset.Ancestors = feedMessageGroup.Asset.Ancestors
	feedMessageGroupSettings.Asset.AncestryPath = feedMessageGroup.Asset.AncestryPath
	feedMessageGroupSettings.Asset.AssetType = "groupssettings.googleapis.com/groupSettings"
	feedMessageGroupSettings.Asset.Name = feedMessageGroup.Asset.Name + "/groupSettings"
	feedMessageGroupSettings.Deleted = feedMessageGroup.Deleted
	if !feedMessageGroup.Deleted {
		groupSettings, err := global.groupsSettingsService.Groups.Get(feedMessageGroup.Asset.Resource.Email).Do()
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "quota") {
				log.Println(glo.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "WARNING",
					Message:            fmt.Sprintf("waiting_on_quota_exceeded"),
					Description:        fmt.Sprintf("GetGroupSettings quota is gone, wait for %d seconds then retry", waitSecOnQuotaExceeded),
					TriggeringPubsubID: global.PubSubID,
				})
				time.Sleep(waitSecOnQuotaExceeded * time.Second)
				return err
			}
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("groupsSettingsService.Groups.Get %v", err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		feedMessageGroupSettings.Asset.Resource = groupSettings
	}
	feedMessageGroupSettings.StepStack = global.stepStack

	feedMessageGroupSettingsJSON, err := json.Marshal(feedMessageGroupSettings)
	if err != nil {
		log.Println(glo.Entry{
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
	publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, global.outputTopicName)
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("global.pubsubPublisherClient.Publish %v %v", &publishRequest, err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	now = time.Now()
	latency := now.Sub(metadata.Timestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(glo.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              fmt.Sprintf("finish %s", feedMessageGroup.Asset.Resource.Email),
		Description:          fmt.Sprintf("groupSettings published to pubsub %s (isdeleted status=%v) %s topic %s ids %v %s", feedMessageGroup.Asset.Resource.Email, feedMessageGroup.Deleted, feedMessageGroup.Asset.Resource.Id, global.outputTopicName, pubsubResponse.MessageIds, string(feedMessageGroupSettingsJSON)),
		Now:                  &now,
		TriggeringPubsubID:   global.PubSubID,
		OriginEventTimestamp: &metadata.Timestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
		AssetInventoryOrigin: feedMessageGroup.Origin,
	})
	return nil
}
