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
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	admin "google.golang.org/api/admin/directory/v1"
)

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
	log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var data []byte
	_, err = base64.StdEncoding.Decode(data, PubSubMessage.Data)
	ram.JSONMarshalIndentPrint(data)

	return nil
}
