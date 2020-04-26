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
	"strings"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"google.golang.org/api/iterator"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// GetTopicList retreive the list of existing pubsub topics
func GetTopicList(ctx context.Context, pubSubPulisherClient *pubsub.PublisherClient, projectID string, topicListPointer *[]string) error {
	var topicList []string
	var listTopicRequest pubsubpb.ListTopicsRequest
	listTopicRequest.Project = fmt.Sprintf("projects/%s", projectID)

	topicsIterator := pubSubPulisherClient.ListTopics(ctx, &listTopicRequest)
	for {
		topic, err := topicsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("topicsIterator.Next: %v", err)
		}
		// log.Printf("topic.Name %v", topic.Name)
		nameParts := strings.Split(topic.Name, "/")
		topicShortName := nameParts[len(nameParts)-1]
		// log.Printf("topicShortName %s", topicShortName)
		topicList = append(topicList, topicShortName)
	}
	// log.Printf("topicList %v", topicList)
	*topicListPointer = topicList
	return nil
}
