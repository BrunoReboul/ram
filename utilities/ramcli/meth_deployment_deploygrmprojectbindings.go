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

func (deployment *Deployment) deployGRMProjectBindings() (err error) {
	projectBindingsDeployment := grm.NewProjectBindingsDeployment()
	projectBindingsDeployment.Core = &deployment.Core
	projectBindingsDeployment.Settings.Roles = deployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Hosting.Project.Roles
	projectBindingsDeployment.Settings.CustomRoles = deployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles
	projectBindingsDeployment.Artifacts.ProjectID = projectBindingsDeployment.Core.SolutionSettings.Hosting.ProjectID

	projectBindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
	err = projectBindingsDeployment.Deploy()
	if err != nil {
		return err
	}

	if projectBindingsDeployment.Core.RamcliServiceAccount != "" {
		projectBindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s", projectBindingsDeployment.Core.RamcliServiceAccount)
		err = projectBindingsDeployment.Deploy()
		if err != nil {
			return err
		}
	}
	return nil
}
