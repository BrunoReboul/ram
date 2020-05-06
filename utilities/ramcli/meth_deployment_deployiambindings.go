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

	"github.com/BrunoReboul/ram/utilities/iamgt"
)

func (deployment *Deployment) deployIAMBindings() (err error) {
	if deployment.Settings.Service.GCB.DeployIAMBindings {
		bindingsDeployment := iamgt.NewBindingsDeployment()
		bindingsDeployment.Core = &deployment.Core
		bindingsDeployment.Settings.Service.IAM = deployment.Settings.Service.GCB.ServiceAccountBindings.IAM

		// target = microservice service account
		bindingsDeployment.Artifacts.ServiceAccountName = fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com",
			deployment.Core.SolutionSettings.Hosting.ProjectID,
			deployment.Core.ServiceName,
			deployment.Core.SolutionSettings.Hosting.ProjectID)
		// Member = Cloud Build service account
		bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
		err = bindingsDeployment.Deploy()
		if err != nil {
			return err
		}
		if bindingsDeployment.Core.RamcliServiceAccount != "" {
			// Member = ram cli service account
			bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s", bindingsDeployment.Core.RamcliServiceAccount)
			err = bindingsDeployment.Deploy()
			if err != nil {
				return err
			}
		}

		// target = app engine servuce account
		bindingsDeployment.Artifacts.ServiceAccountName = fmt.Sprintf("projects/%s/serviceAccounts/%s@appspot.gserviceaccount.com",
			deployment.Core.SolutionSettings.Hosting.ProjectID,
			deployment.Core.SolutionSettings.Hosting.ProjectID)
		// Member = Cloud Build service account
		bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
		err = bindingsDeployment.Deploy()
		if err != nil {
			return err
		}
		if bindingsDeployment.Core.RamcliServiceAccount != "" {
			// Member = ram cli service account
			bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s", bindingsDeployment.Core.RamcliServiceAccount)
			err = bindingsDeployment.Deploy()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
