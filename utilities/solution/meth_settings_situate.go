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

package solution

// Situate set settings from settings based on a given situation
// Situation is the environment name (string)
// Set settings are: folderID, projectID, Stackdriver projectID, Buckets names
func (settings *Settings) Situate(environmentName string) {
	settings.Hosting.OrganizationID = settings.Hosting.OrganizationIDs[environmentName]
	settings.Hosting.FolderID = settings.Hosting.FolderIDs[environmentName]
	settings.Hosting.ProjectID = settings.Hosting.ProjectIDs[environmentName]
	settings.Hosting.Stackdriver.ProjectID = settings.Hosting.Stackdriver.ProjectIDs[environmentName]
	settings.Hosting.GCS.Buckets.CAIExport.Name = settings.Hosting.GCS.Buckets.CAIExport.Names[environmentName]
	settings.Hosting.GCS.Buckets.AssetsJSONFile.Name = settings.Hosting.GCS.Buckets.AssetsJSONFile.Names[environmentName]
}
