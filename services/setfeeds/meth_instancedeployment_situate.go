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
	"fmt"

	"github.com/BrunoReboul/ram/utilities/cai"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	switch instanceDeployment.Settings.Instance.CAI.ContentType {
	case "RESOURCE":
		if len(instanceDeployment.Settings.Instance.CAI.AssetTypes) != 1 {
			return fmt.Errorf("There must be one an only one assetType when ContentType is RESOURCE")
		}
		assetShortName := cai.GetAssetShortTypeName(instanceDeployment.Settings.Instance.CAI.AssetTypes[0])
		instanceDeployment.Artifacts.FeedName = fmt.Sprintf("ram-%s-rce-%s",
			instanceDeployment.Core.EnvironmentName,
			assetShortName)
		instanceDeployment.Artifacts.TopicName = fmt.Sprintf("cai-rces-%s", assetShortName)
		return nil
	case "IAM_POLICY":
		instanceDeployment.Artifacts.FeedName = fmt.Sprintf("ram-%s-iam-policies", instanceDeployment.Core.EnvironmentName)
		instanceDeployment.Artifacts.TopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.IAMPolicies
		return nil
	default:
		return fmt.Errorf("Unsupported instanceDeployment.Settings.Instance.CAI.ContentType %s", instanceDeployment.Settings.Instance.CAI.ContentType)
	}
}
