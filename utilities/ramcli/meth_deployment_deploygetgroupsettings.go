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
	"github.com/BrunoReboul/ram/services/getgroupsettings"
)

func (deployment *Deployment) deployGetGroupSettings() (err error) {
	instanceDeployment := getgroupsettings.NewInstanceDeployment()
	instanceDeployment.Core = &deployment.Core
	err = instanceDeployment.ReadValidate()
	if err != nil {
		return err
	}
	err = instanceDeployment.Situate()
	if err != nil {
		return err
	}
	switch true {
	case deployment.Core.Commands.MakeReleasePipeline:
		deployment.Settings.Service.GCB = instanceDeployment.Settings.Service.GCB
		deployment.Settings.Service.IAM = instanceDeployment.Settings.Service.IAM
		deployment.Settings.Service.GSU = instanceDeployment.Settings.Service.GSU
		deployment.Core.AssetType = ""
		err = deployment.deployInstanceReleasePipeline()
	case deployment.Core.Commands.Deploy:
		if deployment.Core.Commands.Deploy {
			err = instanceDeployment.Deploy()
		}
	}
	if err != nil {
		return err
	}
	return nil
}
