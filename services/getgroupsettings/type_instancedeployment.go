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

package getgroupsettings

import (
	"time"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iamgt"
	"google.golang.org/api/iam/v1"
)

// InstanceDeployment settings and artifacts structure
type InstanceDeployment struct {
	DumpTimestamp time.Time `yaml:"dumpTimestamp"`
	Core          *deploy.Core
	Settings      struct {
		Service struct {
			GSU             gsu.Parameters
			IAM             iamgt.Parameters
			GCB             gcb.Parameters
			GCF             gcf.Parameters
			KeyJSONFileName string `yaml:"keyJSONFileName"`
		}
		Instance struct {
			GCF gcf.Event
			GCI struct {
				SuperAdminEmail string `yaml:"superAdminEmail"`
			}
		}
	}
}

// NewInstanceDeployment create deployment structure with default settings set
func NewInstanceDeployment() *InstanceDeployment {
	var instanceDeployment InstanceDeployment
	instanceDeployment.Settings.Service.GSU.APIList = []string{
		"appengine.googleapis.com",
		"cloudfunctions.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"containerregistry.googleapis.com",
		"iam.googleapis.com",
		"pubsub.googleapis.com",
		"groupssettings.googleapis.com"}
	instanceDeployment.Settings.Service.GSU.APIList = append(deploy.GetCommonAPIlist(), instanceDeployment.Settings.Service.GSU.APIList...)

	instanceDeployment.Settings.Service.IAM.RunRoles.Project = []iam.Role{
		projectRunRole()}
	instanceDeployment.Settings.Service.IAM.DeployRoles.Project = []iam.Role{
		projectDeployCoreRole(),
		iamgt.ProjectDeployExtendedRole()}

	instanceDeployment.Settings.Service.GCB.BuildTimeout = "600s"
	instanceDeployment.Settings.Service.GCB.DeployIAMServiceAccount = true
	instanceDeployment.Settings.Service.GCB.DeployIAMBindings = true
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles = []string{
		projectDeployCoreRole().Title,
		iamgt.ProjectDeployExtendedRole().Title}
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.IAM.RolesOnServiceAccounts = []string{
		"roles/iam.serviceAccountUser"}

	instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles = []string{
		projectRunRole().Title}

	instanceDeployment.Settings.Service.GCF.AvailableMemoryMb = 128
	instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds = 600
	instanceDeployment.Settings.Service.GCF.Timeout = "60s"

	instanceDeployment.Settings.Service.KeyJSONFileName = "key.json"

	return &instanceDeployment
}

func projectRunRole() (role iam.Role) {
	role.Title = "ram_getgroupsettings_run"
	role.Description = "Real-time Asset Monitor get group settings microservice permissions to run"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"iam.serviceAccountKeys.list",
		"iam.serviceAccountKeys.delete",
		"pubsub.topics.create",
		"pubsub.topics.list",
		"pubsub.topics.publish"}
	return role
}

func projectDeployCoreRole() (role iam.Role) {
	role.Title = "ram_getgroupsettings_deploy_core"
	role.Description = "Real-time Asset Monitor get group settings microservice core permissions to deploy"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"iam.serviceAccountKeys.create",
		"pubsub.topics.get",
		"pubsub.topics.create",
		"pubsub.topics.update",
		"cloudfunctions.functions.sourceCodeSet",
		"cloudfunctions.functions.get",
		"cloudfunctions.functions.create",
		"cloudfunctions.functions.update",
		"cloudfunctions.operations.get"}
	return role
}
