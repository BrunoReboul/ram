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

package listgroupmembers

import (
	"fmt"
	"log"

	"google.golang.org/api/iam/v1"
)

// getServiceAccountKey
func (instanceDeployment *InstanceDeployment) getServiceAccountKey() (serviceAccountKey *iam.ServiceAccountKey, err error) {
	log.Printf("%s create a new service account key", instanceDeployment.Core.InstanceName)
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		instanceDeployment.Core.ServiceName,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)
	name := fmt.Sprintf("projects/%s/serviceAccounts/%s",
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		serviceAccountEmail)
	var createServiceAccountKeyRequest iam.CreateServiceAccountKeyRequest

	projectsServiceAccountsKeysService := iam.NewProjectsServiceAccountsKeysService(instanceDeployment.Core.Services.IAMService)
	serviceAccountKey, err = projectsServiceAccountsKeysService.Create(name, &createServiceAccountKeyRequest).Context(instanceDeployment.Core.Ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("iam.NewProjectsServiceAccountsKeysService %v", err)
	}

	return serviceAccountKey, err
}
