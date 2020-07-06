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

package dumpinventory

import (
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iamgt"
	"github.com/BrunoReboul/ram/utilities/sch"
	"google.golang.org/api/iam/v1"
)

// InstanceDeployment settings and artifacts structure
type InstanceDeployment struct {
	DumpTimestamp time.Time `yaml:"dumpTimestamp"`
	Artifacts     struct {
		JobName   string `yaml:"jobName"`
		TopicName string `yaml:"topicName"`
		Schedule  string
	}
	Core     *deploy.Core
	Settings struct {
		Service struct {
			GSU gsu.Parameters
			IAM iamgt.Parameters
			GCB gcb.Parameters
			GCF gcf.Parameters
		}
		Instance struct {
			CAI cai.Parameters
			SCH sch.Parameters
		}
	}
}

// NewInstanceDeployment create deployment structure with default settings set
func NewInstanceDeployment() *InstanceDeployment {
	var instanceDeployment InstanceDeployment
	instanceDeployment.Settings.Service.GSU.APIList = []string{
		"appengine.googleapis.com",
		"cloudbuild.googleapis.com",
		"cloudfunctions.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"containerregistry.googleapis.com",
		"iam.googleapis.com",
		"pubsub.googleapis.com",
		"cloudscheduler.googleapis.com"}
	instanceDeployment.Settings.Service.GSU.APIList = append(deploy.GetCommonAPIlist(), instanceDeployment.Settings.Service.GSU.APIList...)

	instanceDeployment.Settings.Service.IAM.RunRoles.MonitoringOrg = []iam.Role{
		monitoringOrgRunRole()}
	instanceDeployment.Settings.Service.IAM.DeployRoles.MonitoringOrg = []iam.Role{
		monitoringOrgDeployExtendedRole()}
	instanceDeployment.Settings.Service.IAM.DeployRoles.Project = []iam.Role{
		projectDeployCoreRole(),
		iamgt.ProjectDeployExtendedRole()}

	instanceDeployment.Settings.Service.GCB.BuildTimeout = "600s"
	instanceDeployment.Settings.Service.GCB.QueueTTL = "7200s"
	instanceDeployment.Settings.Service.GCB.DeployIAMServiceAccount = true
	instanceDeployment.Settings.Service.GCB.DeployIAMBindings = true
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Monitoring.Org.CustomRoles = []string{
		monitoringOrgDeployExtendedRole().Title}
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles = []string{
		projectDeployCoreRole().Title,
		iamgt.ProjectDeployExtendedRole().Title}
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.IAM.RolesOnServiceAccounts = []string{
		"roles/iam.serviceAccountUser"}

	instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.GRM.Monitoring.Org.CustomRoles = []string{
		monitoringOrgRunRole().Title}

	instanceDeployment.Settings.Service.GCF.AvailableMemoryMb = 128
	instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds = 600
	instanceDeployment.Settings.Service.GCF.Timeout = "60s"

	return &instanceDeployment
}

func monitoringOrgRunRole() (role iam.Role) {
	role.Title = "ram_dumpinventory_monitoring_org_run"
	role.Description = "Real-time Asset Monitor dump inventory microservice permissions to run on monitoring org"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"cloudasset.assets.exportResource",
		"cloudasset.assets.exportIamPolicy"}
	return role
}

func monitoringOrgDeployExtendedRole() (role iam.Role) {
	role.Title = "ram_dumpinventory_monitoring_org_deploy_extended"
	role.Description = "Real-time Asset Monitor dump inventory microservice extended permissions to deploy on monitoring org"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"iam.roles.create",
		"iam.roles.get",
		"iam.roles.update",
		"resourcemanager.organizations.getIamPolicy",
		"resourcemanager.organizations.setIamPolicy"}
	return role
}

func projectDeployCoreRole() (role iam.Role) {
	role.Title = "ram_dumpinventory_deploy_core"
	role.Description = "Real-time Asset Monitor dump inventory microservice core permissions to deploy"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"pubsub.topics.get",
		"pubsub.topics.create",
		"pubsub.topics.update",
		"storage.buckets.get",
		"storage.buckets.create",
		"storage.buckets.update",
		"cloudscheduler.jobs.get",
		"cloudscheduler.jobs.create",
		"cloudfunctions.functions.sourceCodeSet",
		"cloudfunctions.functions.get",
		"cloudfunctions.functions.create",
		"cloudfunctions.functions.update",
		"cloudfunctions.operations.get"}
	return role
}
