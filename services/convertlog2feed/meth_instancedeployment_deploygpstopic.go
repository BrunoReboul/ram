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
	"github.com/BrunoReboul/ram/utilities/gps"
)

func (instanceDeployment *InstanceDeployment) deployGPSTopic() (err error) {
	topicDeployment := gps.NewTopicDeployment()
	topicDeployment.Core = instanceDeployment.Core

	topicDeployment.Settings.TopicName = instanceDeployment.Settings.Instance.GCF.TriggerTopic
	err = topicDeployment.Deploy()
	if err != nil {
		return err
	}

	topicDeployment.Settings.TopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings
	err = topicDeployment.Deploy()
	if err != nil {
		return err
	}

	topicDeployment.Settings.TopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers
	err = topicDeployment.Deploy()
	if err != nil {
		return err
	}

	return nil
}
