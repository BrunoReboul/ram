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
	"github.com/BrunoReboul/ram/utilities/deploy"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
)

// FeedDeployment settings and artifacts structure
type FeedDeployment struct {
	Artifacts struct {
		TopicName    string `yaml:"topicName"`
		FeedName     string `yaml:"feedName"`
		ContentType  assetpb.ContentType
		FeedFullName string `yaml:"FeedFullName"`
	}
	Core     *deploy.Core
	Settings struct {
		Instance struct {
			CAI Parameters
		}
	}
}

// NewFeedDeployment create deployment structure
func NewFeedDeployment() *FeedDeployment {
	return &FeedDeployment{}
}
