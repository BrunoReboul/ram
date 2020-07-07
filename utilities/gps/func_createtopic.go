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
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"github.com/BrunoReboul/ram/utilities/str"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// CreateTopic check if a topic already exist, if not create it
func CreateTopic(ctx context.Context, pubSubPulisherClient *pubsub.PublisherClient, topicListPointer *[]string, topicName string, projectID string) error {
	if str.Find(*topicListPointer, topicName) {
		return nil
	}
	// refresh topic list
	err := GetTopicList(ctx, pubSubPulisherClient, projectID, topicListPointer)
	if err != nil {
		return fmt.Errorf("getTopicList: %v", err)
	}
	if str.Find(*topicListPointer, topicName) {
		return nil
	}
	var topicRequested pubsubpb.Topic
	topicRequested.Name = fmt.Sprintf("projects/%s/topics/%s", projectID, topicName)
	topicRequested.Labels = map[string]string{"name": strings.ToLower(topicName)}

	log.Printf("topicRequested %v", topicRequested)

	topic, err := pubSubPulisherClient.CreateTopic(ctx, &topicRequested)
	if err != nil {
		matched, _ := regexp.Match(`.*AlreadyExists.*`, []byte(err.Error()))
		if !matched {
			return fmt.Errorf("pubSubPulisherClient.CreateTopic: %v", err)
		}
		log.Println("Try to create but already exist:", topicName)
	} else {
		log.Println("Created topic:", topic.Name)
	}
	// refresh topic list
	err = GetTopicList(ctx, pubSubPulisherClient, projectID, topicListPointer)
	if err != nil {
		return fmt.Errorf("getTopicList: %v", err)
	}
	return nil
}
