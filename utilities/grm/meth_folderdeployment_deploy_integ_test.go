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
	"time"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"golang.org/x/oauth2/google"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

// Prerequerisite: must have a sandboxing folder and rights to create delete folders blow that folder
const sanboxesFolderID = "376289220104"
const notActiveFolderName = "ram-grm-integ-test"

func TestIntegFolderDeploy(t *testing.T) {
	type testcases []struct {
		Name            string
		Core            deploy.Core
		WantMsgContains string
		WantError       bool
		WantErrContains string
	}
	var testCases testcases
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

	err := yaml.Unmarshal(yamlBytes, &testCases)
	if err != nil {
		log.Fatalf("Unable to unmarshal yaml test data %v", err)
	}

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("google.FindDefaultCredentials %v", err)
	}

	for _, tc := range testCases {
		tc.Core.Services.CloudresourcemanagerServicev2, err = cloudresourcemanagerv2.NewService(ctx, option.WithCredentials(creds))
		if err != nil {
			log.Fatalln(err)
		}
		if tc.Name == "pendingDeletionFolder" {
			tc.Core.SolutionSettings.Hosting.FolderID = getPendingDeletionFolder(ctx,
				tc.Core.Services.CloudresourcemanagerServicev2,
				sanboxesFolderID)
		}
		t.Run(tc.Name, func(t *testing.T) {
			folderDeployment := NewFolderDeployment()
			folderDeployment.Core = &tc.Core

			var buffer bytes.Buffer
			log.SetOutput(&buffer)
			defer func() {
				log.SetOutput(os.Stderr)
			}()

			err := folderDeployment.Deploy()
			msgString := buffer.String()

			if err != nil {
				if tc.WantError {
					if tc.WantErrContains != "" {
						if !strings.Contains(err.Error(), tc.WantErrContains) {
							t.Logf("want error msg to contains '%s' and got \n'%s'", tc.WantMsgContains, msgString)
						}
					}
				} else {
					t.Errorf("Want NO error and got one %v", err)
				}
			} else {
				if tc.WantError {
					t.Errorf("Want an error and got none")
				}
			}

			if tc.WantMsgContains != "" {
				if !strings.Contains(msgString, tc.WantMsgContains) {
					t.Errorf("want msg to contains '%s' and got \n'%s'", tc.WantMsgContains, msgString)
				}
			}
		})
	}
}

func getPendingDeletionFolder(ctx context.Context, cloudresourcemanagerService *cloudresourcemanagerv2.Service, parent string) (folderID string) {
	foldersService := cloudresourcemanagerService.Folders
	var searchFoldersRequest cloudresourcemanagerv2.SearchFoldersRequest
	var folderName string
	// Avoid creating a not ACTIVE folder if one already exists

	searchFoldersRequest.Query = "NOT lifecycleState=ACTIVE AND parent=folders/" + parent
	searchFoldersResponse, err := foldersService.Search(&searchFoldersRequest).Context(ctx).Do()
	if err != nil {
		log.Fatalf("foldersService.Search existing zomby %v", err)
	}
	if len(searchFoldersResponse.Folders) > 0 {
		// Just use the first found for testing, and extract folder ID
		return getFolderID(searchFoldersResponse.Folders[0].Name)
	}

	// Need to create a not ACTIVE folder for test purpose, aka create delete to have a deletion pending, for one month ...
	searchFoldersRequest.Query = "displayName=" + notActiveFolderName + " AND parent=folders/" + parent
	searchFoldersResponse, err = foldersService.Search(&searchFoldersRequest).Context(ctx).Do()
	if err != nil {
		log.Fatalf("foldersService.Search %s %v", notActiveFolderName, err)
	}
	if len(searchFoldersResponse.Folders) == 0 {
		var folder cloudresourcemanagerv2.Folder
		folder.Parent = "folders/" + parent
		folder.DisplayName = notActiveFolderName
		operation, err := foldersService.Create(&folder).Context(ctx).Do()
		if err != nil {
			// ffo.JSONMarshalIndentPrint(folder)
			log.Fatalf("foldersService.Create %s %v", notActiveFolderName, err)
		}
		name := operation.Name
		for {
			time.Sleep(5 * time.Second)
			for i := 0; i < Retries; i++ {
				operation, err = cloudresourcemanagerService.Operations.Get(name).Context(ctx).Do()
				if err != nil {
					if strings.Contains(err.Error(), "500") && strings.Contains(err.Error(), "backendError") {
						log.Printf("%s ERROR getting operation status, iteration %d, wait 5 sec and retry %v", notActiveFolderName, i, err)
						time.Sleep(5 * time.Second)
					} else {
						log.Fatalf("cloudresourcemanagerService.Operations.Get %s %v", notActiveFolderName, err)
					}
				}
			}
			if err != nil {
				log.Fatalf("cloudresourcemanagerService.Operations.Get %s %v", notActiveFolderName, err)
			}
			if operation.Done {
				break
			}
		}
		if operation.Error != nil {
			log.Fatalf("Function deployment error %v", operation.Error)
		}
		// else err is nil, means the folder has been created
		ffo.JSONMarshalIndentPrint(operation)
		folderName = "TBD"
	} else {
		folderName = searchFoldersResponse.Folders[0].Name

	}
	// Folder found or create at this stage
	_, err = foldersService.Delete(folderName).Context(ctx).Do()
	if err != nil {
		log.Fatalf("foldersService.delete %v", err)
	}
	return getFolderID(folderName)
}

func getFolderID(folderName string) (folderID string) {
	parts := strings.Split(folderName, "/")
	return parts[len(parts)-1]
}
