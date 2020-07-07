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

package grm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/str"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// Deploy use retries on a read-modify-write cycle
func (projectBindingsDeployment *ProjectBindingsDeployment) Deploy() (err error) {
	if len(projectBindingsDeployment.Settings.Roles)+len(projectBindingsDeployment.Settings.CustomRoles) > 0 && projectBindingsDeployment.Artifacts.ProjectID != "" {
		log.Printf("%s grm project bindings", projectBindingsDeployment.Core.InstanceName)
		projectsService := projectBindingsDeployment.Core.Services.CloudresourcemanagerService.Projects
		for i := 0; i < Retries; i++ {
			if i > 0 {
				log.Printf("%s grm retrying a full read-modify-write cycle, iteration %d", projectBindingsDeployment.Core.InstanceName, i)
			}
			// READ
			var policy *cloudresourcemanager.Policy
			var getPolicyOptions cloudresourcemanager.GetPolicyOptions
			var getRequest cloudresourcemanager.GetIamPolicyRequest
			getRequest.Options = &getPolicyOptions
			policy, err = projectsService.GetIamPolicy(projectBindingsDeployment.Artifacts.ProjectID, &getRequest).Context(projectBindingsDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "403") {
					log.Printf("%s grm WARNING impossible to GET project iam policy %v", projectBindingsDeployment.Core.InstanceName, err)
					return nil
				}
				return fmt.Errorf("grm projectsService.GetIamPolicy %s", err)
			}
			// MODIFY
			policyIsToBeUpdated := false
			existingRoles := make([]string, 0)
			for _, binding := range policy.Bindings {
				existingRoles = append(existingRoles, binding.Role)
				if str.Find(projectBindingsDeployment.Settings.Roles, binding.Role) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == projectBindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s grm member %s already have %s on project %s", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.Member, binding.Role, projectBindingsDeployment.Artifacts.ProjectID)
					} else {
						log.Printf("%s grm add member %s to existing %s on project %s", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.Member, binding.Role, projectBindingsDeployment.Artifacts.ProjectID)
						binding.Members = append(binding.Members, projectBindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
				parts := strings.Split(binding.Role, "/")
				customRole := parts[len(parts)-1]
				if str.Find(projectBindingsDeployment.Settings.CustomRoles, customRole) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == projectBindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s grm member %s already have %s on project %s", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.Member, customRole, projectBindingsDeployment.Artifacts.ProjectID)
					} else {
						log.Printf("%s grm add member %s to existing %s on project %s", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.Member, customRole, projectBindingsDeployment.Artifacts.ProjectID)
						binding.Members = append(binding.Members, projectBindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
			}
			for _, role := range projectBindingsDeployment.Settings.Roles {
				if !str.Find(existingRoles, role) {
					var binding cloudresourcemanager.Binding
					binding.Role = role
					binding.Members = []string{projectBindingsDeployment.Artifacts.Member}
					log.Printf("%s grm add new %s with solo member %s on project %s", projectBindingsDeployment.Core.InstanceName, binding.Role, projectBindingsDeployment.Artifacts.Member, projectBindingsDeployment.Artifacts.ProjectID)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			for _, customRole := range projectBindingsDeployment.Settings.CustomRoles {
				role := fmt.Sprintf("projects/%s/roles/%s", projectBindingsDeployment.Artifacts.ProjectID, customRole)
				if !str.Find(existingRoles, role) {
					var binding cloudresourcemanager.Binding
					binding.Role = role
					binding.Members = []string{projectBindingsDeployment.Artifacts.Member}
					log.Printf("%s grm add new %s with solo member %s on project %s", projectBindingsDeployment.Core.InstanceName, binding.Role, projectBindingsDeployment.Artifacts.Member, projectBindingsDeployment.Artifacts.ProjectID)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			// WRITE
			if policyIsToBeUpdated {
				var setRequest cloudresourcemanager.SetIamPolicyRequest
				setRequest.Policy = policy

				var updatedPolicy *cloudresourcemanager.Policy
				updatedPolicy, err = projectsService.SetIamPolicy(projectBindingsDeployment.Artifacts.ProjectID, &setRequest).Context(projectBindingsDeployment.Core.Ctx).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "There were concurrent policy changes") {
						if strings.Contains(err.Error(), "403") {
							log.Printf("%s grm WARNING impossible to SET project iam policy %v", projectBindingsDeployment.Core.InstanceName, err)
							return nil
						}
						return fmt.Errorf("grm projectsService.SetIamPolicy %s", err)
					}
					log.Printf("%s grm there were concurrent policy changes, wait 5 sec and retry a full read-modify-write cycle, iteration %d", projectBindingsDeployment.Core.InstanceName, i)
					time.Sleep(5 * time.Second)
				} else {
					// ram.JSONMarshalIndentPrint(updatedPolicy)
					_ = updatedPolicy
					log.Printf("%s grm project policy updated for %s iteration %d", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.ProjectID, i)
					break
				}
			} else {
				log.Printf("%s grm NO need to update project policy for %s", projectBindingsDeployment.Core.InstanceName, projectBindingsDeployment.Artifacts.ProjectID)
				break
			}
		}
	}
	return err
}
