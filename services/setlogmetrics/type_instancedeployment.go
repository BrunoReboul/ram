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

package setlogmetrics

import (
	"time"

	"github.com/BrunoReboul/ram/utilities/glo"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iamgt"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/logging/v2"
)

// InstanceDeployment settings and artifacts structure
type InstanceDeployment struct {
	DumpTimestamp time.Time `yaml:"dumpTimestamp"`
	Artifacts     struct {
		LogMetric logging.LogMetric
	}
	Core     *deploy.Core
	Settings struct {
		Service struct {
			GSU gsu.Parameters
			IAM iamgt.Parameters
			GCB gcb.Parameters
		}
		Instance struct {
			GLO glo.LogMetricParameters
		}
	}
}

// NewInstanceDeployment create deployment structure with default settings set
func NewInstanceDeployment() *InstanceDeployment {
	var instanceDeployment InstanceDeployment
	instanceDeployment.Settings.Service.GSU.APIList = deploy.GetCommonAPIlist() // No additional APIs than the common list

	instanceDeployment.Settings.Service.IAM.DeployRoles.Project = []iam.Role{
		projectDeployCoreRole(),
		projectDeployExtendedRole()}

	instanceDeployment.Settings.Service.GCB.BuildTimeout = "6000s"
	instanceDeployment.Settings.Service.GCB.DeployIAMServiceAccount = false
	instanceDeployment.Settings.Service.GCB.DeployIAMBindings = false
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.GRM.Hosting.Project.CustomRoles = []string{
		projectDeployCoreRole().Title,
		projectDeployExtendedRole().Title}
	return &instanceDeployment
}

func projectDeployExtendedRole() (role iam.Role) {
	role.Title = "ram_setlogmetrics_deploy_extended"
	role.Description = "Real-time Asset Monitor set log based metrics microservice extended permissions to deploy"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"serviceusage.services.list",
		"serviceusage.services.enable"}
	return role
}

func projectDeployCoreRole() (role iam.Role) {
	role.Title = "ram_setlogmetrics_deploy_core"
	role.Description = "Real-time Asset Monitor set log based metrics microservice core permissions to deploy"
	role.Stage = "GA"
	role.IncludedPermissions = []string{
		"logging.logMetrics.get",
		"logging.logMetrics.create",
		"logging.logMetrics.update"}
	return role
}
