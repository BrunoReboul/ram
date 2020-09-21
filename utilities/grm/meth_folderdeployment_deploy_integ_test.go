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

package grm

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"golang.org/x/oauth2/google"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

// Prerequerisite: must have a sandboxing folder and rights to create delete folders blow that folder
const sanboxesFolderID = "376289220104"
const notActiveFolderName = "ram-grm-integ-test"

func TestIntegFolderDeploy(t *testing.T) {
	type testlist []struct {
		Name            string
		Core            deploy.Core
		WantMsgContains string
		WantError       bool
		WantErrContains string
	}
	var testList testlist
	yamlBytes := []byte(`---
- name: existingFolder
  core:
    solutionsettings:
      hosting:
        folderID: 610996297248
  wantmsgcontains: grm folder found
- name: forbiddenFolder
  core:
    solutionsettings:
      hosting:
        folderID: 610996297249
  wantmsgcontains: grm WARNING impossible to GET folder
- name: wrongFolderID
  core:
    solutionsettings:
      hosting:
        folderID: 01234567891
  wanterror: true
- name: pendingDeletionFolder
  core:
    solutionsettings:
      hosting:
        folderID: TBD
  wanterror: true
  wanterrcontains: is in state DELETE_REQUESTED while it should be ACTIVE`)

	err := yaml.Unmarshal(yamlBytes, &testList)
	if err != nil {
		log.Fatalf("Unable to unmarshal yaml test data %v", err)
	}

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("ERROR - google.FindDefaultCredentials %v", err)
	}

	for _, test := range testList {
		test.Core.Services.CloudresourcemanagerServicev2, err = cloudresourcemanagerv2.NewService(ctx, option.WithCredentials(creds))
		if err != nil {
			log.Fatalln(err)
		}
		if test.Name == "pendingDeletionFolder" {
			test.Core.SolutionSettings.Hosting.FolderID = getPendingDeletionFolder(ctx,
				test.Core.Services.CloudresourcemanagerServicev2,
				sanboxesFolderID)
		}
		t.Run(test.Name, func(t *testing.T) {
			folderDeployment := NewFolderDeployment()
			folderDeployment.Core = &test.Core

			var buffer bytes.Buffer
			log.SetOutput(&buffer)
			defer func() {
				log.SetOutput(os.Stderr)
			}()

			err := folderDeployment.Deploy()
			msgString := buffer.String()

			if err != nil {
				if test.WantError {
					if test.WantErrContains != "" {
						if !strings.Contains(err.Error(), test.WantErrContains) {
							t.Logf("want error msg to contains '%s' and got \n'%s'", test.WantMsgContains, msgString)
						}
					}
				} else {
					t.Errorf("Want NO error and got one %v", err)
				}
			} else {
				if test.WantError {
					t.Errorf("Want an error and got none")
				}
			}

			if test.WantMsgContains != "" {
				if !strings.Contains(msgString, test.WantMsgContains) {
					t.Errorf("want msg to contains '%s' and got \n'%s'", test.WantMsgContains, msgString)
				}
			}
		})
	}
}

func getPendingDeletionFolder(ctx context.Context, cloudresourcemanagerService *cloudresourcemanagerv2.Service, parent string) (folderID string) {
	// foldersService := cloudresourcemanagerService.Folders
	// Avoid creating a not ACTIVE folder if one already exists

	// var searchFoldersRequest cloudresourcemanagerv2.SearchFoldersRequest
	// searchFoldersRequest.Query = "NOT lifecycleState=ACTIVE AND parent=folders/" + parent
	// searchFoldersResponse, err := foldersService.Search(&searchFoldersRequest).Context(ctx).Do()
	// if err != nil {
	// 	log.Fatalf("foldersService.Search %v", err)
	// }
	// if len(searchFoldersResponse.Folders) > 0 {
	// 	// Just use the first found for testing, and extract folder ID
	// 	parts := strings.Split(searchFoldersResponse.Folders[0].Name, "/")
	// 	return parts[len(parts)-1]
	// }

	// Need to create a not ACTIVE folder for test purpose, aka create delete to have a deletion pending, for one month ...
	// searchFoldersRequest.Query = "displayName=" + notActiveFolderName + " AND parent=folders/" + parent
	// searchFoldersResponse, err := foldersService.Search(&searchFoldersRequest).Context(ctx).Do()
	// if err != nil {
	// 	log.Fatalf("foldersService.Search %v", err)
	// }

	// var folderName string
	// if len(searchFoldersResponse.Folders) == 0 {
	// 	log.Fatalln("todo")
	// } else {
	// 	folderName = searchFoldersResponse.Folders[0].Name
	// }
	// folder, err := foldersService.Delete(folderName).Context(ctx).Do()
	// if err != nil {
	// 	log.Fatalf("foldersService.Search %v", err)
	// }

	// folder.DisplayName = notActiveFolderName
	return "336029683157"
}
