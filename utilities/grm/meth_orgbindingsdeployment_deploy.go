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

// Retries is the max number of read-modify-write cycles in case of concurrent policy changes detection
const Retries = 5

// Deploy use retries on a read-modify-write cycle
func (orgBindingsDeployment *OrgBindingsDeployment) Deploy() (err error) {
	if len(orgBindingsDeployment.Settings.Roles)+len(orgBindingsDeployment.Settings.CustomRoles) > 0 && orgBindingsDeployment.Artifacts.OrganizationID != "" {
		log.Printf("%s grm organization bindings", orgBindingsDeployment.Core.InstanceName)
		organizationsService := orgBindingsDeployment.Core.Services.CloudresourcemanagerService.Organizations
		for i := 0; i < Retries; i++ {
			if i > 0 {
				log.Printf("%s grm retrying a full read-modify-write cycle, iteration %d", orgBindingsDeployment.Core.InstanceName, i)
			}
			// READ
			var policy *cloudresourcemanager.Policy
			var getPolicyOptions cloudresourcemanager.GetPolicyOptions
			var getRequest cloudresourcemanager.GetIamPolicyRequest
			getRequest.Options = &getPolicyOptions
			policy, err = organizationsService.GetIamPolicy(fmt.Sprintf("organizations/%s", orgBindingsDeployment.Artifacts.OrganizationID), &getRequest).Context(orgBindingsDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "403") {
					// To not stop on missing org permission, even if checking is not possible
					log.Printf("%s grm WARNING impossible to check nor set organization iam policies due to insufficiant permissions on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.OrganizationID)
					log.Printf("%s grm WARNING moving forward and assuming the required roles have been granted by another chanel on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.OrganizationID)
					return nil
				}
				return fmt.Errorf("grm ram cli organizationsService.GetIamPolicy %v", err)
			}
			// MODIFY
			policyIsToBeUpdated := false
			existingRoles := make([]string, 0)
			for _, binding := range policy.Bindings {
				existingRoles = append(existingRoles, binding.Role)
				if str.Find(orgBindingsDeployment.Settings.Roles, binding.Role) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == orgBindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s grm member %s already have %s on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.Member, binding.Role, orgBindingsDeployment.Artifacts.OrganizationID)
					} else {
						log.Printf("%s grm add member %s to existing %s on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.Member, binding.Role, orgBindingsDeployment.Artifacts.OrganizationID)
						binding.Members = append(binding.Members, orgBindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
				parts := strings.Split(binding.Role, "/")
				customRole := parts[len(parts)-1]
				if str.Find(orgBindingsDeployment.Settings.CustomRoles, customRole) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == orgBindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s grm member %s already have %s on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.Member, customRole, orgBindingsDeployment.Artifacts.OrganizationID)
					} else {
						log.Printf("%s grm add member %s to existing %s on organization %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.Member, customRole, orgBindingsDeployment.Artifacts.OrganizationID)
						binding.Members = append(binding.Members, orgBindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
			}
			for _, role := range orgBindingsDeployment.Settings.Roles {
				if !str.Find(existingRoles, role) {
					var binding cloudresourcemanager.Binding
					binding.Role = role
					binding.Members = []string{orgBindingsDeployment.Artifacts.Member}
					log.Printf("%s grm add new %s with solo member %s on organization %s", orgBindingsDeployment.Core.InstanceName, binding.Role, orgBindingsDeployment.Artifacts.Member, orgBindingsDeployment.Artifacts.OrganizationID)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			for _, customRole := range orgBindingsDeployment.Settings.CustomRoles {
				role := fmt.Sprintf("organizations/%s/roles/%s", orgBindingsDeployment.Artifacts.OrganizationID, customRole)
				if !str.Find(existingRoles, role) {
					var binding cloudresourcemanager.Binding
					binding.Role = role
					binding.Members = []string{orgBindingsDeployment.Artifacts.Member}
					log.Printf("%s grm add new %s with solo member %s on organization %s", orgBindingsDeployment.Core.InstanceName, binding.Role, orgBindingsDeployment.Artifacts.Member, orgBindingsDeployment.Artifacts.OrganizationID)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			// WRITE
			if policyIsToBeUpdated {
				var setRequest cloudresourcemanager.SetIamPolicyRequest
				setRequest.Policy = policy

				var updatedPolicy *cloudresourcemanager.Policy
				updatedPolicy, err = organizationsService.SetIamPolicy(fmt.Sprintf("organizations/%s", orgBindingsDeployment.Artifacts.OrganizationID), &setRequest).Context(orgBindingsDeployment.Core.Ctx).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "There were concurrent policy changes") {
						return fmt.Errorf("organizationsService.SetIamPolicy %v", err)
					}
					log.Printf("%s grm there were concurrent policy changes, wait 5 sec and retry a full read-modify-write cycle, iteration %d", orgBindingsDeployment.Core.InstanceName, i)
					time.Sleep(5 * time.Second)
				} else {
					// ffo.JSONMarshalIndentPrint(updatedPolicy)
					_ = updatedPolicy
					log.Printf("%s grm organization policy updated for %s iteration %d", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.OrganizationID, i)
					break
				}
			} else {
				log.Printf("%s grm NO need to update organization policy for %s", orgBindingsDeployment.Core.InstanceName, orgBindingsDeployment.Artifacts.OrganizationID)
				break
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
