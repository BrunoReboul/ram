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
	"fmt"

	"google.golang.org/api/cloudfunctions/v1"
)

// getUploadURL returns the URL where to upload the cloud function source Zip
func (functionDeployment *FunctionDeployment) getUploadURL() (err error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", functionDeployment.Core.SolutionSettings.Hosting.ProjectID,
		functionDeployment.Core.SolutionSettings.Hosting.GCF.Region)
	// The request body must be empty https://cloud.google.com/functions/docs/reference/rest/v1/projects.locations.functions/generateUploadUrl
	var generateUploadURLRequest cloudfunctions.GenerateUploadUrlRequest
	generateUploadURLResponse, err := functionDeployment.Artifacts.ProjectsLocationsFunctionsService.GenerateUploadUrl(parent,
		&generateUploadURLRequest).Context(functionDeployment.Core.Ctx).Do()
	if err != nil {
		return err
	}
	functionDeployment.Artifacts.CloudFunction.SourceUploadUrl = generateUploadURLResponse.UploadUrl
	return nil
}
