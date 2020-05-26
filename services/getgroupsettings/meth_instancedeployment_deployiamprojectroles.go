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

package getgroupsettings

import (
	"github.com/BrunoReboul/ram/utilities/iamgt"
)

func (instanceDeployment *InstanceDeployment) deployIAMProjectRoles() (err error) {
	if len(instanceDeployment.Settings.Service.IAM.DeployRoles.Project) > 0 {
		projectRolesDeployment := iamgt.NewProjectRolesDeployment()
		projectRolesDeployment.Core = instanceDeployment.Core
		projectRolesDeployment.Settings.Roles = instanceDeployment.Settings.Service.IAM.RunRoles.Project
		projectRolesDeployment.Artifacts.ProjectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
		return projectRolesDeployment.Deploy()
	}
	return nil
}
