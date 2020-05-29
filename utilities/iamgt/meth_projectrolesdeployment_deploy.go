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

package iamgt

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/iam/v1"
)

// Deploy ProjectRolesDeployment
func (projectRolesDeployment *ProjectRolesDeployment) Deploy() (err error) {
	log.Printf("%s iam project roles", projectRolesDeployment.Core.InstanceName)
	projectsRolesService := iam.NewProjectsRolesService(projectRolesDeployment.Core.Services.IAMService)
	for _, customRole := range projectRolesDeployment.Settings.Roles {
		name := fmt.Sprintf("projects/%s/roles/%s",
			projectRolesDeployment.Artifacts.ProjectID, customRole.Title)
		retreivedCustomRole, err := projectsRolesService.Get(name).Context(projectRolesDeployment.Core.Ctx).Do()
		if err != nil {
			if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
				parent := fmt.Sprintf("projects/%s", projectRolesDeployment.Artifacts.ProjectID)
				var createRoleRequest iam.CreateRoleRequest
				createRoleRequest.RoleId = customRole.Title
				createRoleRequest.Role = &customRole
				retreivedCustomRole, err = projectsRolesService.Create(parent, &createRoleRequest).Context(projectRolesDeployment.Core.Ctx).Do()
				if err != nil {
					log.Printf("%s iam WARNING impossible to CREATE custom project roles %v", projectRolesDeployment.Core.InstanceName, err)
					return nil
				}
				log.Printf("%s iam custom project role created %s", projectRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
			} else {
				log.Printf("%s iam WARNING impossible to GET custom project roles %v", projectRolesDeployment.Core.InstanceName, err)
				return nil
			}
		} else {
			log.Printf("%s iam custom project role founded %s", projectRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
			retreivedCustomRole, err = projectsRolesService.Patch(name, &customRole).Context(projectRolesDeployment.Core.Ctx).Do()
			if err != nil {
				log.Printf("%s iam WARNING impossible to PATCH custom project roles %v", projectRolesDeployment.Core.InstanceName, err)
				return nil
			}
			log.Printf("%s iam custom project role patched %s", projectRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
		}
	}
	return nil
}
