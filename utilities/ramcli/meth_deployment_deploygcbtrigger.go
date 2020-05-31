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

package ramcli

import (
	"strings"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/gcb"
)

func (deployment *Deployment) deployGCBTrigger() (err error) {
	triggerDeployment := gcb.NewTriggerDeployment()
	triggerDeployment.Core = &deployment.Core
	triggerDeployment.Settings.Service.GCB = deployment.Settings.Service.GCB
	if deployment.Core.AssetType != "" {
		triggerDeployment.Artifacts.AssetShortTypeName = strings.Replace(cai.GetAssetShortTypeName(deployment.Core.AssetType), "-", "_", -1)
	}
	return triggerDeployment.Deploy()
}
