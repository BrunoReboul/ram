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
	"fmt"

	"google.golang.org/api/cloudfunctions/v1"
)

// GetCloudFunction looks for and existing cloud function
func GetCloudFunction(ctx context.Context, projectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService, projectID, region, instanceName string) (cloudFunction *cloudfunctions.CloudFunction, err error) {
	name := fmt.Sprintf("projects/%s/locations/%s/functions/%s", projectID, region, instanceName)
	cloudFunction, err = projectsLocationsFunctionsService.Get(name).Context(ctx).Do()
	if err != nil {
		return cloudFunction, err
	}
	return cloudFunction, nil
}
