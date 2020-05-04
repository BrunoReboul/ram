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

import "github.com/BrunoReboul/ram/utilities/iamgt"

func (deployment *Deployment) deployIAMMonitoringOrgRole() (err error) {
	if len(deployment.Settings.Service.IAM.DeployRoles.HostingOrg) > 0 {
		orgRoleDeployment := iamgt.NewOrgRolesDeployment()
		orgRoleDeployment.Core = &deployment.Core
		orgRoleDeployment.Settings.Roles = deployment.Settings.Service.IAM.DeployRoles.HostingOrg
		for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
			orgRoleDeployment.Artifacts.OrganizationID = organizationID
			err = orgRoleDeployment.Deploy()
			if err != nil {
				break
			}
		}
		return err
	}
	return nil
}
