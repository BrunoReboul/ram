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

package upload2gcs

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
			GSU gsu.Parameters
			IAM iamgt.Parameters
			GCB gcb.Parameters
			GCF gcf.Parameters
		}
		Instance struct {
			GCF gcf.Event
		}
	}
}

// NewInstanceDeployment create deployment structure with default settings set
func NewInstanceDeployment() *InstanceDeployment {
	var instanceDeployment InstanceDeployment
	instanceDeployment.Settings.Service.GSU.APIList = []string{
		"appengine.googleapis.com",
		"cloudfunctions.googleapis.com",
		"pubsub.googleapis.com"}
	instanceDeployment.Settings.Service.GSU.APIList = append(deploy.GetCommonAPIlist(), instanceDeployment.Settings.Service.GSU.APIList...)

	instanceDeployment.Settings.Service.IAM.RunRoles.MonitoringOrg = []iam.Role{
		monitoringOrgRunRole()}
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

	instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.GRM.Monitoring.Org.CustomRoles = []string{
		monitoringOrgRunRole().Title}

	instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles = []string{
		projectRunRole().Title}
	// Data store permissions are not supported in custom roles
	instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.GRM.Hosting.Project.Roles = []string{
		"roles/datastore.viewer"}

	instanceDeployment.Settings.Service.GCF.AvailableMemoryMb = 256
	instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds = 3600
	instanceDeployment.Settings.Service.GCF.Timeout = "60s"

	return &instanceDeployment
}

func projectRunRole() (role iam.Role) {
	role.Title = "ram_upload2gcs_run"
	role.Description = "Real-time Asset Monitor upload to GCS microservice permissions to run"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"storage.buckets.get",
		"storage.objects.create",
		"storage.objects.delete"}
	return role
}

// used to retreive org, folders, project friendly names when not found in firestore
func monitoringOrgRunRole() (role iam.Role) {
	role.Title = "ram_upload2gcs_monitoring_org_run"
	role.Description = "Real-time Asset Monitor upload to GCS microservice permissions to run on monitoring org"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"resourcemanager.projects.get",
		"resourcemanager.folders.get",
		"resourcemanager.organizations.get"}
	return role
}

func projectDeployCoreRole() (role iam.Role) {
	role.Title = "ram_upload2gcs_deploy_core"
	role.Description = "Real-time Asset Monitor upload to GCS microservice core permissions to deploy"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"pubsub.topics.get",
		"pubsub.topics.create",
		"pubsub.topics.update",
		"storage.buckets.get",
		"storage.buckets.create",
		"storage.buckets.update",
		"cloudfunctions.functions.sourceCodeSet",
		"cloudfunctions.functions.get",
		"cloudfunctions.functions.create",
		"cloudfunctions.functions.update",
		"cloudfunctions.operations.get"}
	return role
}
