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

package itst

import (
	"context"
	"log"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

//GetIntegrationTestsProjectID check that the current project has a name that contains 'ram-build' and return the project ID
// avoiding that integration tests resource creation, deletion occur in a project that is not dedicated to that puppose.
func GetIntegrationTestsProjectID() (projectID string) {
	// OK to use log.Fatal here as this function will not have integration test itself by design
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalln(err)
	}
	cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	projectService := cloudresourcemanagerService.Projects
	project, err := projectService.Get(creds.ProjectID).Context(ctx).Do()
	if err != nil {
		log.Fatalln(err)
	}
	if !strings.Contains(project.Name, "ram-build") {
		log.Fatalln("The project used to run ram integration test MUST have a name that contains 'ram-build'")
	}
	return project.ProjectId
}
