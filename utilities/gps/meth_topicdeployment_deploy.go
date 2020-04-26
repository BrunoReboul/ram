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

package gps

import (
	"fmt"
	"log"
	"strings"

	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
	"google.golang.org/genproto/protobuf/field_mask"
)

// Deploy topic
func (topicDeployment *TopicDeployment) Deploy() (err error) {
	log.Printf("%s gps topic", topicDeployment.Core.InstanceName)
	topicName := fmt.Sprintf("projects/%s/topics/%s",
		topicDeployment.Core.SolutionSettings.Hosting.ProjectID,
		topicDeployment.Settings.TopicName)
	var getTopicRequest pubsubpb.GetTopicRequest
	getTopicRequest.Topic = topicName
	topicNotFound := false
	nameLabelToBeUpdated := false
	topic, err := topicDeployment.Core.Services.PubsubPublisherClient.GetTopic(topicDeployment.Core.Ctx, &getTopicRequest)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			topicNotFound = true
		} else {
			return err
		}
	} else {
		if topic.Labels != nil {
			if topic.Labels["name"] != strings.ToLower(topicDeployment.Settings.TopicName) {
				nameLabelToBeUpdated = true
			}
		} else {
			nameLabelToBeUpdated = true
		}
	}
	if topicNotFound {
		var topicToCreate pubsubpb.Topic
		topicToCreate.Name = topicName
		topicToCreate.Labels = map[string]string{"name": strings.ToLower(topicDeployment.Settings.TopicName)}
		_, err = topicDeployment.Core.Services.PubsubPublisherClient.CreateTopic(topicDeployment.Core.Ctx, &topicToCreate)
		if err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "alreadyexists") {
				return err
			}
			log.Printf("%s gps try to create topic but already exist %s", topicDeployment.Core.InstanceName, topicDeployment.Settings.TopicName)
		}
		log.Printf("%s gps topic created %s", topicDeployment.Core.InstanceName, topicDeployment.Settings.TopicName)
	} else {
		if nameLabelToBeUpdated {
			var fieldMask field_mask.FieldMask
			fieldMask.Paths = []string{"labels"}
			topic.Labels = map[string]string{"name": strings.ToLower(topicDeployment.Settings.TopicName)}
			var updateTopicRequest pubsubpb.UpdateTopicRequest
			updateTopicRequest.Topic = topic
			updateTopicRequest.UpdateMask = &fieldMask
			_, err = topicDeployment.Core.Services.PubsubPublisherClient.UpdateTopic(topicDeployment.Core.Ctx, &updateTopicRequest)
			if err != nil {
				return err
			}
			log.Printf("%s gps topic found, label updated %s", topicDeployment.Core.InstanceName, topicDeployment.Settings.TopicName)
		} else {
			log.Printf("%s gps topic found %s", topicDeployment.Core.InstanceName, topicDeployment.Settings.TopicName)
		}
	}
	return nil
}
