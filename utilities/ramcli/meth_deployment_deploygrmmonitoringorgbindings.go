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
	"fmt"

	"github.com/BrunoReboul/ram/utilities/grm"
)

func (deployment *Deployment) deployGRMMonitoringOrgBindings() (err error) {
	orgBindingsDeployment := grm.NewOrgBindingsDeployment()
	orgBindingsDeployment.Core = &deployment.Core
	orgBindingsDeployment.Settings.Roles = deployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Monitoring.Org.Roles
	orgBindingsDeployment.Settings.CustomRoles = deployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Monitoring.Org.CustomRoles
	for _, organizationID := range orgBindingsDeployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		orgBindingsDeployment.Artifacts.OrganizationID = organizationID

		orgBindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
		err = orgBindingsDeployment.Deploy()
		if err != nil {
			return err
		}

		if orgBindingsDeployment.Core.RamcliServiceAccount != "" {
			orgBindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s", orgBindingsDeployment.Core.RamcliServiceAccount)
			err = orgBindingsDeployment.Deploy()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
