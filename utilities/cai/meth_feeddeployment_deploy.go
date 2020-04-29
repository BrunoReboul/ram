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

	"github.com/BrunoReboul/ram/utilities/ram"

	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
)

// Deploy get-create resource feeds, get-create-update the iam policies feed
func (feedDeployment *FeedDeployment) Deploy() (err error) {
	log.Printf("%s cai cloud asset inventory feed", feedDeployment.Core.InstanceName)
	var getFeedRequest assetpb.GetFeedRequest
	getFeedRequest.Name = fmt.Sprintf("%s/feeds/%s",
		feedDeployment.Settings.Instance.CAI.Parent, feedDeployment.Artifacts.FeedName)
	feed, err := feedDeployment.Core.Services.AssetClient.GetFeed(feedDeployment.Core.Ctx, &getFeedRequest)
	if err != nil {
		return err
	}
	ram.JSONMarshalIndentPrint(feed)
	return nil
}
