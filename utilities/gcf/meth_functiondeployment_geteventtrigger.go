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

package gcf

import (
	"fmt"

	"google.golang.org/api/cloudfunctions/v1"
)

func (functionDeployment *FunctionDeployment) getEventTrigger() (eventTrigger *cloudfunctions.EventTrigger, err error) {
	var failurePolicy cloudfunctions.FailurePolicy
	retry := cloudfunctions.Retry{}
	failurePolicy.Retry = &retry

	switch functionDeployment.Settings.Service.GCF.FunctionType {
	case "backgroundPubSub":
		var evtTrigger cloudfunctions.EventTrigger
		evtTrigger.EventType = "google.pubsub.topic.publish"
		evtTrigger.Resource = fmt.Sprintf("projects/%s/topics/%s", functionDeployment.Core.SolutionSettings.Hosting.ProjectID, functionDeployment.Settings.Instance.GCF.TriggerTopic)
		evtTrigger.Service = "pubsub.googleapis.com"
		evtTrigger.FailurePolicy = &failurePolicy
		return &evtTrigger, nil
	case "backgroundGCS":
		var evtTrigger cloudfunctions.EventTrigger
		evtTrigger.EventType = "google.storage.object.finalize"
		evtTrigger.Resource = fmt.Sprintf("projects/_/buckets/%s", functionDeployment.Settings.Instance.GCF.BucketName)
		evtTrigger.Service = "storage.googleapis.com"
		evtTrigger.FailurePolicy = &failurePolicy
		return &evtTrigger, nil
	default:
		return eventTrigger, fmt.Errorf("functionType provided not managed: %s", functionDeployment.Settings.Service.GCF.FunctionType)
	}
}
