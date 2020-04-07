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

// Package ramdeploy is a utility to hel deploying ram
package ramdeploy

// SolutionSettings settings common to all services / all instances
type SolutionSettings struct {
	Hosting struct {
		BillingAccountID string            `yaml:"billingAccountID"`
		FolderIDs        map[string]string `yaml:"folderIDs"`
		ProjectIDs       map[string]string `yaml:"projectIDs"`
		StackDriver      struct {
			ProjectIDs map[string]string `yaml:"projectIDs"`
		}
		Repository struct {
			Name string
		}
		GAE struct {
			Region string
		}
		GCF struct {
			Region string
		}
		GCS struct {
			CAIExport struct {
				BucketNames map[string]string `yaml:"bucketNames"`
			}
			AssetsJSONFile struct {
				BucketNames map[string]string `yaml:"bucketNames"`
			}
		}
		BQ struct {
			DatasetName string `yaml:"datasetName"`
			Location    string
		}
		PupSub struct {
			TopicName struct {
				IAM                 string `yaml:"IAM"`
				RAMViolation        string `yaml:"RAMViolation"`
				RAMComplianceStatus string `yaml:"RAMComplianceStatus"`
			}
		}
	}
	Monitoring struct {
		OrganizationIDList      []string          `yaml:"organizationIDList"`
		DirectoryCustomerIDList map[string]string `yaml:"directoryCustomerIDList"`
		AssetTypeList           struct {
			IAM       []string `yaml:"iam"`
			Resources []string `yaml:"resources"`
		}
	}
}
