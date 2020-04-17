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

// GetUploadURL returns the URL where to upload the cloud function source Zip
func (goGCFArtifacts *GoGCFArtifacts) GetUploadURL(projectID, region string) (err error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, region)
	// The request body must be empty https://cloud.google.com/functions/docs/reference/rest/v1/projects.locations.functions/generateUploadUrl
	var generateUploadURLRequest cloudfunctions.GenerateUploadUrlRequest
	generateUploadURLResponse, err := goGCFArtifacts.ProjectsLocationsFunctionsService.GenerateUploadUrl(parent, &generateUploadURLRequest).Context(goGCFArtifacts.Ctx).Do()
	if err != nil {
		return err
	}
	goGCFArtifacts.CloudFunction.SourceUploadUrl = generateUploadURLResponse.UploadUrl
	return nil
}
