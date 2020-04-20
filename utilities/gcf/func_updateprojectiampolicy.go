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
	"github.com/BrunoReboul/ram/utilities/ram"

	"google.golang.org/api/cloudresourcemanager/v1"
)

// UpdateProjectIAMPolicy creates the service account to be used by a cloud function
func (goGCFArtifacts *GoGCFArtifacts) UpdateProjectIAMPolicy() (err error) {
	projectsService := cloudresourcemanager.NewProjectsService(goGCFArtifacts.CloudresourcemanagerService)
	var policyOptions cloudresourcemanager.GetPolicyOptions
	// policyOptions.RequestedPolicyVersion = 2
	var request cloudresourcemanager.GetIamPolicyRequest
	request.Options = &policyOptions
	// resource := fmt.Sprintf("projects/%s", goGCFArtifacts.ProjectID)
	policy, err := projectsService.GetIamPolicy(goGCFArtifacts.ProjectID, &request).Context(goGCFArtifacts.Ctx).Do()
	if err != nil {
		return err
	}

	// member := fmt.Sprintf("serviceAccount:", goGCFArtifacts.CloudFunction.ServiceAccountEmail)

	// for _, binding := range policy.Bindings {

	// }
	ram.JSONMarshalIndentPrint(&policy)
	return nil
}
