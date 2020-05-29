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

package cai

import (
	"fmt"
	"log"
	"strings"

	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
)

// Deploy get-create resource feeds, get-create-update the iam policies feed
func (feedDeployment *FeedDeployment) Deploy() (err error) {
	log.Printf("%s cai cloud asset inventory feed", feedDeployment.Core.InstanceName)
	feedDeployment.Artifacts.FeedFullName = fmt.Sprintf("%s/feeds/%s",
		feedDeployment.Settings.Instance.CAI.Parent, feedDeployment.Artifacts.FeedName)
	var getFeedRequest assetpb.GetFeedRequest
	getFeedRequest.Name = feedDeployment.Artifacts.FeedFullName
	feedFound := true
	// GET
	feed, err := feedDeployment.Core.Services.AssetClient.GetFeed(feedDeployment.Core.Ctx, &getFeedRequest)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			feedFound = false
		} else {
			return fmt.Errorf("AssetClient.GetFeed %v", err)
		}
	}
	if feedFound {
		switch feed.ContentType {
		case assetpb.ContentType_RESOURCE:
			log.Printf("%s cai feed found. will NOT be updated as type is RESOURCE %s", feedDeployment.Core.InstanceName, feed.Name)
			return nil
		case assetpb.ContentType_IAM_POLICY:
			return feedDeployment.updateFeed(feed)
		default:
			return fmt.Errorf("Feed found of unmanged ContentType %v", feed.ContentType)
		}
	} else {
		return feedDeployment.createFeed()
	}
}

func (feedDeployment *FeedDeployment) createFeed() (err error) {

	var pubsubDestination assetpb.PubsubDestination
	pubsubDestination.Topic = fmt.Sprintf("projects/%s/topics/%s",
		feedDeployment.Core.SolutionSettings.Hosting.ProjectID,
		feedDeployment.Artifacts.TopicName)

	var feedOutputConfigPubsubDestination assetpb.FeedOutputConfig_PubsubDestination
	feedOutputConfigPubsubDestination.PubsubDestination = &pubsubDestination

	var feedOuputConfig assetpb.FeedOutputConfig
	feedOuputConfig.Destination = &feedOutputConfigPubsubDestination

	var feedToCreate assetpb.Feed
	// The field name must be empty and it will be generated
	feedToCreate.ContentType = feedDeployment.Artifacts.ContentType
	feedToCreate.AssetTypes = feedDeployment.Settings.Instance.CAI.AssetTypes
	feedToCreate.FeedOutputConfig = &feedOuputConfig

	var createFeedRequest assetpb.CreateFeedRequest
	createFeedRequest.Parent = feedDeployment.Settings.Instance.CAI.Parent
	// This is the client-assigned asset feed identifier and it needs to be unique under a specific parent project/folder/organization
	createFeedRequest.FeedId = feedDeployment.Artifacts.FeedName
	createFeedRequest.Feed = &feedToCreate
	// ram.JSONMarshalIndentPrint(&createFeedRequest)

	feed, err := feedDeployment.Core.Services.AssetClient.CreateFeed(feedDeployment.Core.Ctx, &createFeedRequest)
	if err != nil {
		return fmt.Errorf("AssetClient.CreateFeed %v", err)
	}
	log.Printf("%s cai feed created %s", feedDeployment.Core.InstanceName, feed.Name)
	return nil
}

func (feedDeployment *FeedDeployment) updateFeed(existingFeed *assetpb.Feed) (err error) {
	log.Printf("%s cai feed found, starting update as type is IAM_POLICY %s", feedDeployment.Core.InstanceName, existingFeed.Name)
	existingFeed.AssetTypes = feedDeployment.Settings.Instance.CAI.AssetTypes

	var updateFeedRequest assetpb.UpdateFeedRequest
	updateFeedRequest.Feed = existingFeed

	feed, err := feedDeployment.Core.Services.AssetClient.UpdateFeed(feedDeployment.Core.Ctx, &updateFeedRequest)
	if err != nil {
		return fmt.Errorf("AssetClient.UpdateFeed %v", err)
	}
	log.Printf("%s cai feed updated %s", feedDeployment.Core.InstanceName, feed.Name)
	return nil
}
