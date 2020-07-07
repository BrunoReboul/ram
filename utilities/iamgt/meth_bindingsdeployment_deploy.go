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
	"time"

	"github.com/BrunoReboul/ram/utilities/str"
	"google.golang.org/api/iam/v1"
)

// Retries is the max number of read-modify-write cycles in case of concurrent policy changes detection
const Retries = 5

// Deploy BindingsDeployment use retries on a read-modify-write cycle
func (bindingsDeployment *BindingsDeployment) Deploy() (err error) {
	if len(bindingsDeployment.Settings.Service.IAM.RolesOnServiceAccounts) > 0 {
		log.Printf("%s iam service accounts bindings", bindingsDeployment.Core.InstanceName)
		projectsServiceAccountsService := bindingsDeployment.Core.Services.IAMService.Projects.ServiceAccounts
		for i := 0; i < Retries; i++ {
			if i > 0 {
				log.Printf("%s iam retrying a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
			}
			// READ
			var policy *iam.Policy
			policy, err = projectsServiceAccountsService.GetIamPolicy(bindingsDeployment.Artifacts.ServiceAccountName).Context(bindingsDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "403") {
					log.Printf("%s iam WARNING impossible to GET service account iam policy %v", bindingsDeployment.Core.InstanceName, err)
					return nil
				}
				return fmt.Errorf("iam projectsServiceAccountsService.GetIamPolicy %s", err)
			}
			// MODIFY
			policyIsToBeUpdated := false
			existingRoles := make([]string, 0)
			for _, binding := range policy.Bindings {
				existingRoles = append(existingRoles, binding.Role)
				if str.Find(bindingsDeployment.Settings.Service.IAM.RolesOnServiceAccounts, binding.Role) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == bindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s iam member %s already have %s on service account %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, bindingsDeployment.Artifacts.ServiceAccountName)
					} else {
						log.Printf("%s iam add member %s to existing %s on service account %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, bindingsDeployment.Artifacts.ServiceAccountName)
						binding.Members = append(binding.Members, bindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
			}
			for _, role := range bindingsDeployment.Settings.Service.IAM.RolesOnServiceAccounts {
				if !str.Find(existingRoles, role) {
					var binding iam.Binding
					binding.Role = role
					binding.Members = []string{bindingsDeployment.Artifacts.Member}
					log.Printf("%s iam add new %s with solo member %s on service account %s", bindingsDeployment.Core.InstanceName, binding.Role, bindingsDeployment.Artifacts.Member, bindingsDeployment.Artifacts.ServiceAccountName)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			// WRITE
			if policyIsToBeUpdated {
				var setRequest iam.SetIamPolicyRequest
				setRequest.Policy = policy

				var updatedPolicy *iam.Policy
				updatedPolicy, err = projectsServiceAccountsService.SetIamPolicy(bindingsDeployment.Artifacts.ServiceAccountName, &setRequest).Context(bindingsDeployment.Core.Ctx).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "There were concurrent policy changes") {
						if strings.Contains(err.Error(), "403") {
							log.Printf("%s iam WARNING impossible to SET service account iam policy %v", bindingsDeployment.Core.InstanceName, err)
							return nil
						}
						return fmt.Errorf("iam projectsServiceAccountsService.SetIamPolicy %s", err)
					}
					log.Printf("%s iam there were concurrent policy changes, wait 5 sec and retry a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
					time.Sleep(5 * time.Second)
				} else {
					// ram.JSONMarshalIndentPrint(updatedPolicy)
					_ = updatedPolicy
					log.Printf("%s iam policy updated for service account %s iteration %d", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.ServiceAccountName, i)
					break
				}
			} else {
				log.Printf("%s iam NO need to update iam policy for service account %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.ServiceAccountName)
				break
			}
		}
	}
	return err
}
