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

// Package gcf helps to deploy cloud functions
package gcf

import (
	"context"

	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

// GoGCFArtifacts struct to deploy a Go CloudFunction
type GoGCFArtifacts struct {
	Ctx                               context.Context `yaml:"-"`
	CloudFunction                     cloudfunctions.CloudFunction
	CloudFunctionZipFullPath          string
	CloudresourcemanagerService       *cloudresourcemanager.Service `yaml:"-"`
	Dump                              bool
	EnvironmentName                   string
	GoVersion                         string
	IAMService                        *iam.Service `yaml:"-"`
	InstanceName                      string
	OperationsService                 *cloudfunctions.OperationsService                 `yaml:"-"`
	ProjectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService `yaml:"-"`
	ProjectID                         string
	RAMVersion                        string
	Region                            string
	RepositoryPath                    string
	ServiceName                       string
	ZipFiles                          map[string]string
}
