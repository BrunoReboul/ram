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

package stream2bq

import (
	"fmt"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	instanceDeployment.Settings.Service.GCF.FunctionType = "backgroundPubSub"
	instanceDeployment.Settings.Service.GCF.Description = fmt.Sprintf("%s streams from pubsub topic %s to bigquery table %s",
		instanceDeployment.Core.InstanceName,
		instanceDeployment.Settings.Instance.GCF.TriggerTopic,
		instanceDeployment.Settings.Instance.Bigquery.TableName)
	return nil
}
