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
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/ram"
)

// Deploy an instance trigger
func (InstanceTriggerDeployment *InstanceTriggerDeployment) Deploy() (err error) {
	var deployment ram.Deployment

	apiDeployment := gsu.NewAPIDeployment()
	apiDeployment.Core = InstanceTriggerDeployment.Core
	apiDeployment.Artifacts.ServiceusageService = InstanceTriggerDeployment.Artifacts.ServiceusageService
	apiDeployment.Settings.Service.GSU = InstanceTriggerDeployment.Settings.Service.GSU
	deployment = apiDeployment
	err = deployment.Deploy()
	if err != nil {
		return err
	}

	triggerDeployment := gcb.NewTriggerDeployment()
	triggerDeployment.Core = InstanceTriggerDeployment.Core
	triggerDeployment.Artifacts.ProjectsTriggersService = InstanceTriggerDeployment.Artifacts.ProjectsTriggersService
	triggerDeployment.Settings.Service.GCB = InstanceTriggerDeployment.Settings.Service.GCB
	deployment = triggerDeployment
	err = deployment.Deploy()
	if err != nil {
		return err
	}

	return nil
}
