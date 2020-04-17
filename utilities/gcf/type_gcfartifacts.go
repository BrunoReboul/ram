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
	"context"

	"google.golang.org/api/cloudfunctions/v1"
)

// CloudFunctionZipFullPath const
const CloudFunctionZipFullPath = "./cloud_function_source.zip"

// GoGCFArtifacts struct to deploy a Go CloudFunction
type GoGCFArtifacts struct {
	Ctx                               context.Context `yaml:"-"`
	GoVersion                         string
	RAMVersion                        string
	RepositoryPath                    string
	EnvironmentName                   string
	InstanceName                      string
	Dump                              bool
	ProjectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService `yaml:"-"`
	OperationsService                 *cloudfunctions.OperationsService                 `yaml:"-"`
	CloudFunction                     cloudfunctions.CloudFunction
	Location                          string
	ZipFiles                          map[string]string
}
