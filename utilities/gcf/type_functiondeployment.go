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

package gcf

import (
	"github.com/BrunoReboul/ram/utilities/deploy"
	"google.golang.org/api/cloudfunctions/v1"
)

// FunctionDeployment settings and artifacts structure
type FunctionDeployment struct {
	Artifacts struct {
		OperationsService                 *cloudfunctions.OperationsService                 `yaml:"-"`
		ProjectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService `yaml:"-"`
		CloudFunction                     cloudfunctions.CloudFunction
		CloudFunctionZipFullPath          string
		InstanceDeploymentYAMLContent     string
		ZipFiles                          map[string]string
	}
	Core     *deploy.Core
	Settings struct {
		Service struct {
			GCF Parameters
		}
		Instance struct {
			GCF Event
		}
	}
}

// NewFunctionDeployment create deployment structure
func NewFunctionDeployment() *FunctionDeployment {
	return &FunctionDeployment{}
}
