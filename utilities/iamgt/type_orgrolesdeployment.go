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
	"github.com/BrunoReboul/ram/utilities/deploy"
	"google.golang.org/api/iam/v1"
)

// OrgRolesDeployment struct
type OrgRolesDeployment struct {
	Core      *deploy.Core
	Artifacts struct {
		OrganizationID string
	}
	Settings struct {
		Roles []iam.Role
	}
}

// NewOrgRolesDeployment create deployment structure
func NewOrgRolesDeployment() *OrgRolesDeployment {
	return &OrgRolesDeployment{}
}

// role.Name = "ram_cli_org_core"
// role.Title = role.Name
// role.Description = "Real-time Asset Monitor cli mandatory permissions at organization level"
// role.IncludedPermissions = []string{
// 	"cloudasset.feeds.get",
// 	"cloudasset.feeds.create",
// 	"cloudasset.feeds.update",
// 	"cloudasset.assets.exportResource",
// 	"cloudasset.assets.exportIamPolicy"}
// orgRoleDeployment.Artifacts.Roles = append(orgRoleDeployment.Artifacts.Roles, role)

// role.Name = "ram_cli_org_optional"
// role.Title = role.Name
// role.Description = "Real-time Asset Monitor cli complementary permissions at organization level"
// role.IncludedPermissions = []string{
// 	"iam.roles.get",
// 	"iam.roles.create",
// 	"iam.roles.update",
// 	"resourcemanager.organizations.getIamPolicy",
// 	"resourcemanager.organizations.setIamPolicy"}
// orgRoleDeployment.Artifacts.Roles = append(orgRoleDeployment.Artifacts.Roles, role)

// role.Name = "ram_cli_folder_core"
// role.Title = role.Name
// role.Description = "Real-time Asset Monitor cli complementary permissions at organization level"
// role.IncludedPermissions = []string{
// 	"cloudbuild.builds.list",
// 	"cloudbuild.builds.create",
// 	"pubsub.topics.get",
// 	"pubsub.topics.create",
// 	"pubsub.topics.update"}
// orgRoleDeployment.Artifacts.Roles = append(orgRoleDeployment.Artifacts.Roles, role)

// role.Name = "ram_cli_folder_optional"
// role.Title = role.Name
// role.Description = "Real-time Asset Monitor cli complementary permissions at organization level"
// role.IncludedPermissions = []string{
// 	"resourcemanager.folders.get",
// 	"resourcemanager.projects.get",
// 	"resourcemanager.projects.create",
// 	"resourcemanager.projects.createBillingAssignment",
// 	"serviceusage.services.list",
// 	"serviceusage.services.enable",
// 	"resourcemanager.projects.getIamPolicy",
// 	"resourcemanager.projects.setIamPolicy",
// 	"iam.serviceAccounts.get",
// 	"iam.serviceAccounts.create",
// 	"iam.serviceAccounts.getIamPolicy",
// 	"iam.serviceAccounts.setIamPolicy",
// 	"source.repos.get",
// 	"source.repos.create"}
// orgRoleDeployment.Artifacts.Roles = append(orgRoleDeployment.Artifacts.Roles, role)
