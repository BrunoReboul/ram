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

package listgroupmembers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BrunoReboul/ram/utilities/aut"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/logging"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"
	admin "google.golang.org/api/admin/directory/v1"
)

// Global variable to deal with GroupsListCall Pages constraint: no possible to pass variable to the function in pages()
// https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
var ancestors []string
var ctx context.Context
var environment string
var groupAssetName string
var groupEmail string
var instanceName string
var logEventEveryXPubSubMsg uint64
var microserviceName string
var origin string
var outputTopicName string
var pubSubClient *pubsub.Client
var pubSubErrNumber uint64
var pubSubID string
var pubSubMsgNumber uint64
var stepStack logging.Steps
var timestamp time.Time

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	collectionID            string
	ctx                     context.Context
	dirAdminService         *admin.Service
	environment             string
	firestoreClient         *firestore.Client
	instanceName            string
	logEventEveryXPubSubMsg uint64
	maxResultsPerPage       int64 // API Max = 200
	microserviceName        string
	outputTopicName         string
	projectID               string
	pubSubClient            *pubsub.Client
	pubsubID                string
	retryTimeOutSeconds     int64
	step                    logging.Step
	stepStack               logging.Steps
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
	global.logEventEveryXPubSubMsg = instanceDeployment.Settings.Service.LogEventEveryXPubSubMsg
	global.outputTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	global.maxResultsPerPage = instanceDeployment.Settings.Service.MaxResultsPerPage
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
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
		[]string{admin.AdminDirectoryGroupMemberReadonlyScope},
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
	global.pubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("pubsub.NewClient %v", err),
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
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("pubsub_id no available metadata.FromContext: %v", err),
			TriggeringPubsubID: global.pubsubID,
		})
		return err
	}
	global.pubsubID = metadata.EventID
	parts := strings.Split(metadata.Resource.Name, "/")
	global.step = logging.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.pubsubID),
		StepTimestamp: metadata.Timestamp,
	}

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(logging.Entry{
		MicroserviceName:           global.microserviceName,
		InstanceName:               global.instanceName,
		Environment:                global.environment,
		Severity:                   "NOTICE",
		Message:                    "start",
		TriggeringPubsubID:         global.pubsubID,
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
			TriggeringPubsubID:         global.pubsubID,
			TriggeringPubsubAgeSeconds: d.Seconds(),
			TriggeringPubsubTimestamp:  &metadata.Timestamp,
			Now:                        &now,
		})
		return nil
	}

	// Pass data to global variables to deal with func browseGroup
	ctx = global.ctx
	logEventEveryXPubSubMsg = global.logEventEveryXPubSubMsg
	pubSubClient = global.pubSubClient
	outputTopicName = global.outputTopicName
	timestamp = metadata.Timestamp
	pubSubID = global.pubsubID
	microserviceName = global.microserviceName
	instanceName = global.instanceName
	environment = global.environment

	var feedMessageGroup cai.FeedMessageGroup
	err = json.Unmarshal(PubSubMessage.Data, &feedMessageGroup)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)", global.pubsubID)
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        fmt.Sprintf("json.Unmarshal(pubSubMessage.Data, &feedMessageGroup) %v %v", PubSubMessage.Data, err),
			TriggeringPubsubID: global.pubsubID,
		})
		return nil
	}
	global.stepStack = append(feedMessageGroup.StepStack, global.step)
	stepStack = global.stepStack

	pubSubMsgNumber = 0
	groupAssetName = feedMessageGroup.Asset.Name
	groupEmail = feedMessageGroup.Asset.Resource.Email
	// First ancestor is my parent
	ancestors = []string{fmt.Sprintf("groups/%s", feedMessageGroup.Asset.Resource.Id)}
	// Next ancestors are my parent ancestors
	for _, ancestor := range feedMessageGroup.Asset.Ancestors {
		ancestors = append(ancestors, ancestor)
	}
	origin = feedMessageGroup.Origin
	if feedMessageGroup.Deleted {
		// retreive members from cache
		err = browseFeedMessageGroupMembersFromCache(global)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("browseFeedMessageGroupMembersFromCache(global) %v", err),
				TriggeringPubsubID: global.pubsubID,
			})
			return err
		}
	} else {
		// retreive members from admin SDK
		// pages function except just the name of the callback function. Not an invocation of the function
		err = global.dirAdminService.Members.List(feedMessageGroup.Asset.Resource.Id).MaxResults(global.maxResultsPerPage).Pages(ctx, browseMembers)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("dirAdminService.Members.List %v", err),
				TriggeringPubsubID: global.pubsubID,
			})
			return err
		}
	}
	now = time.Now()
	latency := now.Sub(global.step.StepTimestamp)
	latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
	log.Println(logging.Entry{
		MicroserviceName:     global.microserviceName,
		InstanceName:         global.instanceName,
		Environment:          global.environment,
		Severity:             "NOTICE",
		Message:              fmt.Sprintf("finish %s %d", feedMessageGroup.Asset.Resource.Email, pubSubMsgNumber),
		Description:          fmt.Sprintf("Group %s %s isDeleted %v Number of members published to pubsub topic %s: %d", feedMessageGroup.Asset.Resource.Email, feedMessageGroup.Asset.Resource.Id, feedMessageGroup.Deleted, outputTopicName, pubSubMsgNumber),
		Now:                  &now,
		TriggeringPubsubID:   global.pubsubID,
		OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
		LatencySeconds:       latency.Seconds(),
		LatencyE2ESeconds:    latencyE2E.Seconds(),
		StepStack:            global.stepStack,
	})
	return nil
}

func browseFeedMessageGroupMembersFromCache(global *Global) (err error) {
	var waitgroup sync.WaitGroup
	var documentSnap *firestore.DocumentSnapshot
	var feedMessageMember cai.FeedMessageMember
	topic := pubSubClient.Topic(outputTopicName)
	assets := global.firestoreClient.Collection(global.collectionID)
	query := assets.Where(
		"asset.assetType", "==", "www.googleapis.com/admin/directory/members").Where(
		"asset.resource.groupEmail", "==", strings.ToLower(groupEmail))
	iter := query.Documents(global.ctx)
	defer iter.Stop()
	for {
		documentSnap, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "WARNING",
				Message:            "browseFeedMessageGroupMembersFromCache",
				Description:        fmt.Sprintf("log and move to next iter.Next() %v", err),
				TriggeringPubsubID: global.pubsubID,
			})
		} else {
			if documentSnap.Exists() {
				err = documentSnap.DataTo(&feedMessageMember)
				if err != nil {
					log.Println(logging.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            "browseFeedMessageGroupMembersFromCache",
						Description:        fmt.Sprintf("log and move to next documentSnap.DataTo %v", err),
						TriggeringPubsubID: global.pubsubID,
					})
				} else {
					feedMessageMember.Deleted = true
					feedMessageMember.Window.StartTime = timestamp
					feedMessageMember.Origin = origin
					feedMessageMemberJSON, err := json.Marshal(feedMessageMember)
					if err != nil {
						log.Println(logging.Entry{
							MicroserviceName:   global.microserviceName,
							InstanceName:       global.instanceName,
							Environment:        global.environment,
							Severity:           "WARNING",
							Message:            "browseFeedMessageGroupMembersFromCache",
							Description:        fmt.Sprintf("log and move to next %s json.Marshal(feedMessageMember) %v", feedMessageMember.Asset.Name, err),
							TriggeringPubsubID: global.pubsubID,
						})
					} else {
						pubSubMessage := &pubsub.Message{
							Data: feedMessageMemberJSON,
						}
						publishResult := topic.Publish(ctx, pubSubMessage)
						waitgroup.Add(1)
						go gps.GetPublishCallResult(ctx,
							publishResult,
							&waitgroup,
							feedMessageMember.Asset.Name,
							&pubSubErrNumber,
							&pubSubMsgNumber,
							logEventEveryXPubSubMsg,
							pubSubID,
							microserviceName,
							instanceName,
							environment)
					}
				}
			} else {
				log.Println(logging.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "WARNING",
					Message:            "browseFeedMessageGroupMembersFromCache",
					Description:        fmt.Sprintf("log and move to next document does not exists %s", documentSnap.Ref.Path),
					TriggeringPubsubID: global.pubsubID,
				})
			}
		}
	}
	return nil
}

// browseMembers is executed for each page returning a set of members
// A non-nil error returned will halt the iteration
// the only accepted parameter is groups: https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
// so, it use global variables to this package
func browseMembers(members *admin.Members) error {
	var waitgroup sync.WaitGroup
	topic := pubSubClient.Topic(outputTopicName)
	for _, member := range members.Members {
		var feedMessageMember cai.FeedMessageMember
		feedMessageMember.Window.StartTime = timestamp
		feedMessageMember.Origin = origin
		feedMessageMember.Asset.Ancestors = ancestors
		feedMessageMember.Asset.AncestryPath = groupAssetName
		feedMessageMember.Asset.AssetType = "www.googleapis.com/admin/directory/members"
		feedMessageMember.Asset.Name = groupAssetName + "/members/" + member.Id
		feedMessageMember.Asset.Resource.GroupEmail = groupEmail
		feedMessageMember.Asset.Resource.MemberEmail = member.Email
		feedMessageMember.Asset.Resource.ID = member.Id
		feedMessageMember.Asset.Resource.Kind = member.Kind
		feedMessageMember.Asset.Resource.Role = member.Role
		feedMessageMember.Asset.Resource.Type = member.Type
		feedMessageMember.StepStack = stepStack
		feedMessageMemberJSON, err := json.Marshal(feedMessageMember)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName:   microserviceName,
				InstanceName:       instanceName,
				Environment:        environment,
				Severity:           "WARNING",
				Message:            "browseMembers",
				Description:        fmt.Sprintf("log and move to next %s json.Marshal(feedMessageMember) %v", member.Email, err),
				TriggeringPubsubID: pubSubID,
			})
		} else {
			pubSubMessage := &pubsub.Message{
				Data: feedMessageMemberJSON,
			}
			publishResult := topic.Publish(ctx, pubSubMessage)
			waitgroup.Add(1)
			go gps.GetPublishCallResult(ctx,
				publishResult, &waitgroup,
				groupAssetName+"/"+member.Email,
				&pubSubErrNumber,
				&pubSubMsgNumber,
				logEventEveryXPubSubMsg,
				pubSubID,
				microserviceName,
				instanceName,
				environment)
		}
	}
	return nil
}
