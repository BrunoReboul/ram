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
	"os"
	"sync"
	"time"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/option"

	"cloud.google.com/go/pubsub"
	admin "google.golang.org/api/admin/directory/v1"
)

// Global variable to deal with GroupsListCall Pages constraint: no possible to pass variable to the function in pages()
// https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
var ancestors []string
var ctx context.Context
var groupAssetName string
var groupEmail string
var logEventEveryXPubSubMsg uint64
var pubSubClient *pubsub.Client
var outputTopicName string
var pubSubErrNumber uint64
var pubSubMsgNumber uint64
var timestamp time.Time
var origin string

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                     context.Context
	dirAdminService         *admin.Service
	initFailed              bool
	logEventEveryXPubSubMsg uint64
	maxResultsPerPage       int64 // API Max = 200
	outputTopicName         string
	pubSubClient            *pubsub.Client
	retryTimeOutSeconds     int64
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
	global.logEventEveryXPubSubMsg = instanceDeployment.Settings.Service.LogEventEveryXPubSubMsg
	global.outputTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	global.maxResultsPerPage = instanceDeployment.Settings.Service.MaxResultsPerPage
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	keyJSONFilePath := "./" + instanceDeployment.Settings.Service.KeyJSONFileName
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	serviceAccountEmail := os.Getenv("FUNCTION_IDENTITY")

	if clientOption, ok = ram.GetClientOptionAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, projectID, gciAdminUserToImpersonate, []string{admin.AdminDirectoryGroupMemberReadonlyScope}); !ok {
		return
	}
	global.dirAdminService, err = admin.NewService(ctx, clientOption)
	if err != nil {
		log.Printf("ERROR - admin.NewService: %v", err)
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
	ctx = global.ctx
	logEventEveryXPubSubMsg = global.logEventEveryXPubSubMsg
	pubSubClient = global.pubSubClient
	outputTopicName = global.outputTopicName
	timestamp = metadata.Timestamp

	var feedMessageGroup ram.FeedMessageGroup
	err = json.Unmarshal(PubSubMessage.Data, &feedMessageGroup)
	if err != nil {
		log.Println("ERROR - json.Unmarshal(pubSubMessage.Data, &feedMessageGroup)")
		return nil // NO RETRY
	}

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
	// pages function except just the name of the callback function. Not an invocation of the function
	err = global.dirAdminService.Members.List(feedMessageGroup.Asset.Resource.Id).MaxResults(global.maxResultsPerPage).Pages(ctx, browseMembers)
	if err != nil {
		return fmt.Errorf("dirAdminService.Members.List: %v", err) // RETRY
	}
	log.Printf("Completed - Group %s %s Number of members published to pubsub topic %s: %d", feedMessageGroup.Asset.Resource.Id, feedMessageGroup.Asset.Resource.Email, outputTopicName, pubSubMsgNumber)
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
		var feedMessageMember ram.FeedMessageMember
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
		feedMessageMemberJSON, err := json.Marshal(feedMessageMember)
		if err != nil {
			log.Printf("ERROR - %s json.Marshal(feedMessageMember): %v", member.Email, err)
		} else {
			pubSubMessage := &pubsub.Message{
				Data: feedMessageMemberJSON,
			}
			publishResult := topic.Publish(ctx, pubSubMessage)
			waitgroup.Add(1)
			go ram.GetPublishCallResult(ctx, publishResult, &waitgroup, groupAssetName+"/"+member.Email, &pubSubErrNumber, &pubSubMsgNumber, logEventEveryXPubSubMsg)
		}
	}
	return nil
}
