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

package setfeeds

import (
	"log"
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/gps"

	"github.com/BrunoReboul/ram/utilities/gsu"
)

// Deploy a service instance
func (instanceDeployment *InstanceDeployment) Deploy() (err error) {
	start := time.Now()
	// Extended project
	if err = instanceDeployment.deployGSUAPI(); err != nil {
		return err
	}
	// Core project
	if err = instanceDeployment.deployGPSTopic(); err != nil {
		return err
	}
	// Core monitoring orgs
	if err = instanceDeployment.deployCAIFeed(); err != nil {
		return err
	}
	log.Printf("%s done in %v minutes", instanceDeployment.Core.InstanceName, time.Since(start).Minutes())
	return nil
}

func (instanceDeployment *InstanceDeployment) deployGSUAPI() (err error) {
	apiDeployment := gsu.NewAPIDeployment()
	apiDeployment.Core = instanceDeployment.Core
	apiDeployment.Settings.Service.GSU = instanceDeployment.Settings.Service.GSU
	return apiDeployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployGPSTopic() (err error) {
	topicDeployment := gps.NewTopicDeployment()
	topicDeployment.Core = instanceDeployment.Core
	topicDeployment.Settings.TopicName = instanceDeployment.Artifacts.TopicName
	return topicDeployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployCAIFeed() (err error) {
	topicDeployment := cai.NewFeedDeployment()
	topicDeployment.Core = instanceDeployment.Core
	topicDeployment.Artifacts.FeedName = instanceDeployment.Artifacts.FeedName
	topicDeployment.Artifacts.TopicName = instanceDeployment.Artifacts.TopicName
	topicDeployment.Artifacts.ContentType = instanceDeployment.Artifacts.ContentType
	topicDeployment.Settings.Instance.CAI = instanceDeployment.Settings.Instance.CAI
	return topicDeployment.Deploy()
}
