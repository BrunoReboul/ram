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

// GET API requires both 'iam.roles.get' and 'iam.roles.list' permissions

// Deploy OrgRolesDeployment
func (orgRolesDeployment *OrgRolesDeployment) Deploy() (err error) {
	log.Printf("%s iam organization roles", orgRolesDeployment.Core.InstanceName)
	organizationsRolesService := iam.NewOrganizationsRolesService(orgRolesDeployment.Core.Services.IAMService)
	for _, customRole := range orgRolesDeployment.Settings.Roles {
		name := fmt.Sprintf("organizations/%s/roles/%s",
			orgRolesDeployment.Artifacts.OrganizationID, customRole.Title)
		log.Printf("name %s", name)
		retreivedCustomRole, err := organizationsRolesService.Get(name).Context(orgRolesDeployment.Core.Ctx).Do()
		if err != nil {
			if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
				parent := fmt.Sprintf("organizations/%s", orgRolesDeployment.Artifacts.OrganizationID)
				var createRoleRequest iam.CreateRoleRequest
				createRoleRequest.Role = &customRole
				createRoleRequest.RoleId = customRole.Title
				retreivedCustomRole, err = organizationsRolesService.Create(parent, &createRoleRequest).Context(orgRolesDeployment.Core.Ctx).Do()
				if err != nil {
					log.Printf("%s iam WARNING impossible to CREATE custom organization roles %v", orgRolesDeployment.Core.InstanceName, err)
					return nil
				}
				log.Printf("%s iam custom org role created %s", orgRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
			} else {
				log.Printf("%s iam WARNING impossible to GET custom organization roles %v", orgRolesDeployment.Core.InstanceName, err)
				return nil
			}
		} else {
			log.Printf("%s iam found custom org role %s", orgRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
			retreivedCustomRole, err = organizationsRolesService.Patch(name, &customRole).Context(orgRolesDeployment.Core.Ctx).Do()
			if err != nil {
				log.Printf("%s iam WARNING impossible to PATCH custom organization roles %v", orgRolesDeployment.Core.InstanceName, err)
				return nil
			}
			log.Printf("%s iam custom org role patched %s", orgRolesDeployment.Core.InstanceName, retreivedCustomRole.Name)
		}
	}
	return nil
}
