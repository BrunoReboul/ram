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

	"github.com/BrunoReboul/ram/utilities/ram"

	"google.golang.org/api/cloudresourcemanager/v1"
)

// Retries is the max number of read-modify-write cycles in case of concurrent policy changes detection
const Retries = 5

// Deploy BindingsDeployment use retries on a read-modify-write cycle
func (bindingsDeployment *BindingsDeployment) Deploy() (err error) {
	log.Printf("%s grm Resource Manager bindings on 1)RAM project 2) targeted organizations", bindingsDeployment.Core.InstanceName)
	if err = bindingsDeployment.deployRAMProjectBindings(); err != nil {
		return err
	}
	if err = bindingsDeployment.deployOrganizationsBindings(); err != nil {
		return err
	}
	return nil
}

func (bindingsDeployment *BindingsDeployment) deployRAMProjectBindings() (err error) {
	projectsService := bindingsDeployment.Artifacts.CloudresourcemanagerService.Projects
	for i := 0; i < Retries; i++ {
		if i > 0 {
			log.Printf("%s retrying a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
		}
		// READ
		var policy *cloudresourcemanager.Policy
		var getPolicyOptions cloudresourcemanager.GetPolicyOptions
		var getRequest cloudresourcemanager.GetIamPolicyRequest
		getRequest.Options = &getPolicyOptions
		policy, err = projectsService.GetIamPolicy(bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID, &getRequest).Context(bindingsDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
		// MODIFY
		policyIsToBeUpdated := false
		existingRoles := make([]string, 0)
		for _, binding := range policy.Bindings {
			existingRoles = append(existingRoles, binding.Role)
			if ram.Find(bindingsDeployment.Settings.Service.GRM.RolesOnRAMProject, binding.Role) {
				isAlreadyMemberOf := false
				for _, member := range binding.Members {
					if member == bindingsDeployment.Artifacts.Member {
						isAlreadyMemberOf = true
					}
				}
				if isAlreadyMemberOf {
					log.Printf("%s member %s already have %s on project %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID)
				} else {
					log.Printf("%s add member %s to existing %s on project %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID)
					binding.Members = append(binding.Members, bindingsDeployment.Artifacts.Member)
					policyIsToBeUpdated = true
				}
			}
		}
		for _, role := range bindingsDeployment.Settings.Service.GRM.RolesOnRAMProject {
			if !ram.Find(existingRoles, role) {
				var binding cloudresourcemanager.Binding
				binding.Role = role
				binding.Members = []string{bindingsDeployment.Artifacts.Member}
				log.Printf("%s add new %s with solo member %s on project %s", bindingsDeployment.Core.InstanceName, binding.Role, bindingsDeployment.Artifacts.Member, bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID)
				policy.Bindings = append(policy.Bindings, &binding)
				policyIsToBeUpdated = true
			}
		}
		// WRITE
		if policyIsToBeUpdated {
			var setRequest cloudresourcemanager.SetIamPolicyRequest
			setRequest.Policy = policy

			var updatedPolicy *cloudresourcemanager.Policy
			updatedPolicy, err = projectsService.SetIamPolicy(bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID, &setRequest).Context(bindingsDeployment.Core.Ctx).Do()
			if err != nil {
				if !strings.Contains(err.Error(), "There were concurrent policy changes") {
					return err
				}
				log.Printf("%s There were concurrent policy changes, wait 5 sec and retry a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
				time.Sleep(5 * time.Second)
			} else {
				// ram.JSONMarshalIndentPrint(updatedPolicy)
				_ = updatedPolicy
				log.Printf("%s iam policy updated for project %s iteration %d", bindingsDeployment.Core.InstanceName, bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID, i)
				break
			}
		} else {
			log.Printf("%s NO need to update iam policy for project %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Core.SolutionSettings.Hosting.ProjectID)
			break
		}
	}
	return err
}

func (bindingsDeployment *BindingsDeployment) deployOrganizationsBindings() (err error) {
	organizationsService := bindingsDeployment.Artifacts.CloudresourcemanagerService.Organizations
	for _, organizationID := range bindingsDeployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		for i := 0; i < Retries; i++ {
			if i > 0 {
				log.Printf("%s retrying a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
			}
			// READ
			var policy *cloudresourcemanager.Policy
			var getPolicyOptions cloudresourcemanager.GetPolicyOptions
			var getRequest cloudresourcemanager.GetIamPolicyRequest
			getRequest.Options = &getPolicyOptions
			policy, err = organizationsService.GetIamPolicy(fmt.Sprintf("organizations/%s", organizationID), &getRequest).Context(bindingsDeployment.Core.Ctx).Do()
			if err != nil {
				return err
			}
			// MODIFY
			policyIsToBeUpdated := false
			existingRoles := make([]string, 0)
			for _, binding := range policy.Bindings {
				existingRoles = append(existingRoles, binding.Role)
				if ram.Find(bindingsDeployment.Settings.Service.GRM.RolesOnOrganizations, binding.Role) {
					isAlreadyMemberOf := false
					for _, member := range binding.Members {
						if member == bindingsDeployment.Artifacts.Member {
							isAlreadyMemberOf = true
						}
					}
					if isAlreadyMemberOf {
						log.Printf("%s member %s already have %s on organization %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, organizationID)
					} else {
						log.Printf("%s add member %s to existing %s on organization %s", bindingsDeployment.Core.InstanceName, bindingsDeployment.Artifacts.Member, binding.Role, organizationID)
						binding.Members = append(binding.Members, bindingsDeployment.Artifacts.Member)
						policyIsToBeUpdated = true
					}
				}
			}
			for _, role := range bindingsDeployment.Settings.Service.GRM.RolesOnOrganizations {
				if !ram.Find(existingRoles, role) {
					var binding cloudresourcemanager.Binding
					binding.Role = role
					binding.Members = []string{bindingsDeployment.Artifacts.Member}
					log.Printf("%s add new %s with solo member %s on organization %s", bindingsDeployment.Core.InstanceName, binding.Role, bindingsDeployment.Artifacts.Member, organizationID)
					policy.Bindings = append(policy.Bindings, &binding)
					policyIsToBeUpdated = true
				}
			}
			// WRITE
			if policyIsToBeUpdated {
				var setRequest cloudresourcemanager.SetIamPolicyRequest
				setRequest.Policy = policy

				var updatedPolicy *cloudresourcemanager.Policy
				updatedPolicy, err = organizationsService.SetIamPolicy(fmt.Sprintf("organizations/%s", organizationID), &setRequest).Context(bindingsDeployment.Core.Ctx).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "There were concurrent policy changes") {
						return err
					}
					log.Printf("%s There were concurrent policy changes, wait 5 sec and retry a full read-modify-write cycle, iteration %d", bindingsDeployment.Core.InstanceName, i)
					time.Sleep(5 * time.Second)
				} else {
					// ram.JSONMarshalIndentPrint(updatedPolicy)
					_ = updatedPolicy
					log.Printf("%s iam policy updated for organization %s iteration %d", bindingsDeployment.Core.InstanceName, organizationID, i)
					break
				}
			} else {
				log.Printf("%s NO need to update iam policy for organization %s", bindingsDeployment.Core.InstanceName, organizationID)
				break
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
