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

package ramcli

import (
	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gsu"

	"github.com/BrunoReboul/ram/utilities/gcb"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/serviceusage/v1"
)

// InstanceTriggerDeployment structure
type InstanceTriggerDeployment struct {
	Artifacts struct {
		CloudresourcemanagerService *cloudresourcemanager.Service       `yaml:"-"`
		IAMService                  *iam.Service                        `yaml:"-"`
		ProjectsTriggersService     *cloudbuild.ProjectsTriggersService `yaml:"-"`
		ServiceusageService         *serviceusage.Service               `yaml:"-"`
	}
	Core     deploy.Core
	Settings struct {
		Service struct {
			GCB gcb.Parameters
			GSU gsu.Parameters
		}
	}
}

// NewInstanceTrigger create deployment structure
func NewInstanceTrigger() *InstanceTriggerDeployment {
	return &InstanceTriggerDeployment{}
}
