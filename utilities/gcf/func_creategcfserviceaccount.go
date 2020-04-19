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

package gcf

import (
	"fmt"
	"strings"

	"google.golang.org/api/iam/v1"
)

// CreateGCFServiceAccount creates the service account to be used by a cloud function
func (goGCFArtifacts *GoGCFArtifacts) CreateGCFServiceAccount() (err error) {
	projectServiceAccountService := iam.NewProjectsServiceAccountsService(goGCFArtifacts.IAMService)
	projectName := fmt.Sprintf("projects/%s", goGCFArtifacts.ProjectID)
	serviceAccountName := fmt.Sprintf("%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectName, goGCFArtifacts.ServiceName, goGCFArtifacts.ProjectID)
	serviceAccountPtr, err := projectServiceAccountService.Get(serviceAccountName).Context(goGCFArtifacts.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
			var serviceAccount iam.ServiceAccount
			serviceAccount.DisplayName = fmt.Sprintf("RAM %s", goGCFArtifacts.ServiceName)
			serviceAccount.Description = fmt.Sprintf("Solution: Real-time Asset Monitor, microservice: %s", goGCFArtifacts.ServiceName)
			var request iam.CreateServiceAccountRequest
			request.AccountId = goGCFArtifacts.ServiceName
			request.ServiceAccount = &serviceAccount
			serviceAccountPtr, err = projectServiceAccountService.Create(projectName, &request).Context(goGCFArtifacts.Ctx).Do()
			if err != nil {
				// deal with parallel deployments
				if strings.Contains(err.Error(), "alreadyExists") {
					serviceAccountPtr, err = projectServiceAccountService.Get(serviceAccountName).Context(goGCFArtifacts.Ctx).Do()
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		} else {
			return err
		}
	}
	goGCFArtifacts.CloudFunction.ServiceAccountEmail = serviceAccountPtr.Email
	return nil
}
