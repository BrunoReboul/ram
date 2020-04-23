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

package iam

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/iam/v1"
)

// Deploy ServiceaccountDeployment
func (serviceaccountDeployment *ServiceaccountDeployment) Deploy() (err error) {
	projectName := fmt.Sprintf("projects/%s", serviceaccountDeployment.Core.SolutionSettings.Hosting.ProjectID)
	serviceAccountName := fmt.Sprintf("%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectName, serviceaccountDeployment.Core.ServiceName, serviceaccountDeployment.Core.SolutionSettings.Hosting.ProjectID)
	projectServiceAccountService := serviceaccountDeployment.Artifacts.IAMService.Projects.ServiceAccounts
	retreivedServiceAccount, err := projectServiceAccountService.Get(serviceAccountName).Context(serviceaccountDeployment.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
			var serviceAccount iam.ServiceAccount
			serviceAccount.DisplayName = fmt.Sprintf("RAM %s", serviceaccountDeployment.Core.ServiceName)
			serviceAccount.Description = fmt.Sprintf("Solution: Real-time Asset Monitor, microservice: %s", serviceaccountDeployment.Core.ServiceName)
			var request iam.CreateServiceAccountRequest
			request.AccountId = serviceaccountDeployment.Core.ServiceName
			request.ServiceAccount = &serviceAccount
			retreivedServiceAccount, err = projectServiceAccountService.Create(projectName, &request).Context(serviceaccountDeployment.Core.Ctx).Do()
			if err != nil {
				// deal with parallel deployments
				if strings.Contains(err.Error(), "alreadyExists") {
					retreivedServiceAccount, err = projectServiceAccountService.Get(serviceAccountName).Context(serviceaccountDeployment.Core.Ctx).Do()
					if err != nil {
						return err
					}
					log.Printf("%s iam eventually found service account %s", serviceaccountDeployment.Core.InstanceName, retreivedServiceAccount.Email)
				} else {
					return err
				}
			}
			log.Printf("%s iam service account created %s", serviceaccountDeployment.Core.InstanceName, retreivedServiceAccount.Email)
		} else {
			return err
		}
	} else {
		log.Printf("%s iam found service account %s", serviceaccountDeployment.Core.InstanceName, retreivedServiceAccount.Email)
	}
	return nil
}
