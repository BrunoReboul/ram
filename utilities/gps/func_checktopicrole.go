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

	"cloud.google.com/go/iam"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// CheckTopicRole check if a role exist on a topic
func CheckTopicRole(ctx context.Context, pubSubPulisherClient *pubsub.PublisherClient, topicName string, member string, role iam.RoleName) (err error) {
	var getTopicRequest pubsubpb.GetTopicRequest
	getTopicRequest.Topic = topicName
	topic, err := pubSubPulisherClient.GetTopic(ctx, &getTopicRequest)
	if err != nil {
		return fmt.Errorf("pubSubPulisherClient.GetTopic %s %v", topicName, err)
	}

	iamHandle := pubSubPulisherClient.TopicIAM(topic)
	policy, err := iamHandle.Policy(ctx)
	if err != nil {
		return fmt.Errorf("iamHandle.Policy %v", err)
	}

	if !policy.HasRole(member, role) {
		return fmt.Errorf("Missing role %s for member %s on topic %s", role, member, topicName)
	}
	return nil
}
