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

package setfeeds

import (
	"time"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gsu"
)

// InstanceDeployment settings and artifacts structure
type InstanceDeployment struct {
	DumpTimestamp time.Time `yaml:"dumpTimestamp"`
	Artifacts     struct {
		TopicName string `yaml:"topicName"`
		FeedName  string `yaml:"feedName"`
	}
	Core     *deploy.Core
	Settings struct {
		Service struct {
			GSU gsu.Parameters
			GCB gcb.Parameters
		}
		Instance struct {
			CAI cai.Parameters
		}
	}
}

// NewInstanceDeployment create deployment structure with default settings set
func NewInstanceDeployment() *InstanceDeployment {
	var instanceDeployment InstanceDeployment
	instanceDeployment.Settings.Service.GCB.BuildTimeout = "6000s"
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.ResourceManager.RolesOnOrganizations = []string{
		"roles/cloudasset.owner"}
	instanceDeployment.Settings.Service.GCB.ServiceAccountBindings.ResourceManager.RolesOnRAMProject = []string{
		"roles/serviceusage.serviceUsageAdmin",
		"roles/resourcemanager.projectIamAdmin",
		"roles/pubsub.editor"}
	instanceDeployment.Settings.Service.GSU.APIList = []string{
		"cloudasset.googleapis.com",
		"cloudbuild.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"containerregistry.googleapis.com",
		"iam.googleapis.com",
		"pubsub.googleapis.com"}
	instanceDeployment.Settings.Service.GSU.APIList = append(deploy.GetCommonAPIlist(), instanceDeployment.Settings.Service.GSU.APIList...)
	return &instanceDeployment
}
