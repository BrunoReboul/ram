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

import (
	"log"
	"strconv"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestSituate(t *testing.T) {
	type testlist []struct {
		Name        string
		Settings    Settings
		Environment string
		Want        map[string]string
	}
	var testList testlist

	yamlBytes := []byte(`---
- name: set1
  settings:
    hosting:
      organizationIDs:
        dev: "111111111111"
        prd: "222222222222"
      folderIDs:
        dev: "333333333333"
        prd: "444444444444"
      projectIDs:
        dev: blabladev
        prd: blablaprd
      stackdriver:
        projectIDs:
          dev: blabladev
          prd: blablaprd
      gcs:
        buckets:
          CAIExport:
            names:
              dev: blabla-cai-exports-dev
              prd: blabla-cai-exports-prd
          assetsJSONFile:
            names:
              dev: blabla-assets-json-dev
              prd: blabla-assets-json-prd
  environment: dev
  want:
    organizationID: 111111111111
    folderID: 333333333333
    projectID: blabladev
    stackdriverPjID: blabladev
    CAIExportBuccketName: blabla-cai-exports-dev
    CAIExportBuccketDeleteAgeInDays: 3
    assetsJSONBuccketName: blabla-assets-json-dev
    assetsJSONBuccketDeleteAgeInDays: 365
    GCBQueueTTL: 7200s
- name: set2
  settings:
    hosting:
      gcs:
        buckets:
          CAIExport:
            deleteAgeInDays: 99
          assetsJSONFile:
            deleteAgeInDays: 9
      gcb:
        queueTtl: 123s
  environment: dev
  want:
    CAIExportBuccketDeleteAgeInDays: 99
    assetsJSONBuccketDeleteAgeInDays: 9
    GCBQueueTTL: 123s`)

	err := yaml.Unmarshal(yamlBytes, &testList)
	if err != nil {
		log.Fatalf("Unable to unmarshal yaml test data %v", err)
	}

	for _, test := range testList {
		test.Settings.Situate(test.Environment)
		for key, wantedValue := range test.Want {
			testName := test.Name + "-" + key
			t.Run(testName, func(t *testing.T) {
				// t.Logf("%s", testName)
				switch key {
				case "organizationID":
					if wantedValue != test.Settings.Hosting.OrganizationID {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.OrganizationID)
					}
				case "folderID":
					if wantedValue != test.Settings.Hosting.FolderID {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.FolderID)
					}
				case "projectID":
					if wantedValue != test.Settings.Hosting.ProjectID {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.ProjectID)
					}
				case "stackdriverPjID":
					if wantedValue != test.Settings.Hosting.Stackdriver.ProjectID {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.Stackdriver.ProjectID)
					}
				case "CAIExportBuccketName":
					if wantedValue != test.Settings.Hosting.GCS.Buckets.CAIExport.Name {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.GCS.Buckets.CAIExport.Name)
					}
				case "CAIExportBuccketDeleteAgeInDays":
					wantedValueInt64, err := strconv.ParseInt(wantedValue, 10, 64)
					if err != nil {
						t.Errorf("Wanted value cannot be convected to int64 '%s'", wantedValue)
					}
					if wantedValueInt64 != test.Settings.Hosting.GCS.Buckets.CAIExport.DeleteAgeInDays {
						t.Errorf("Want %s '%d' got '%d'", key, wantedValueInt64, test.Settings.Hosting.GCS.Buckets.CAIExport.DeleteAgeInDays)
					}
				case "assetsJSONBuccketName":
					if wantedValue != test.Settings.Hosting.GCS.Buckets.AssetsJSONFile.Name {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.GCS.Buckets.AssetsJSONFile.Name)
					}
				case "assetsJSONBuccketDeleteAgeInDays":
					wantedValueInt64, err := strconv.ParseInt(wantedValue, 10, 64)
					if err != nil {
						t.Errorf("Wanted value cannot be convected to int64 '%s'", wantedValue)
					}
					if wantedValueInt64 != test.Settings.Hosting.GCS.Buckets.AssetsJSONFile.DeleteAgeInDays {
						t.Errorf("Want %s '%d' got '%d'", key, wantedValueInt64, test.Settings.Hosting.GCS.Buckets.AssetsJSONFile.DeleteAgeInDays)
					}
				case "GCBQueueTTL":
					if wantedValue != test.Settings.Hosting.GCB.QueueTTL {
						t.Errorf("Want %s '%s' got '%s'", key, wantedValue, test.Settings.Hosting.GCB.QueueTTL)
					}
				default:
					t.Errorf("Unmanaged key '%s'", key)
				}
			})
		}
	}
}
