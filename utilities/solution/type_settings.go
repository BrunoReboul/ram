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

// Settings settings common to all services / all instances
type Settings struct {
	Hosting struct {
		OrganizationID   string            `yaml:"organizationID,omitempty"`
		OrganizationIDs  map[string]string `yaml:"organizationIDs"`
		BillingAccountID string            `yaml:"billingAccountID"`
		FolderID         string            `yaml:"folderID,omitempty"`
		FolderIDs        map[string]string `yaml:"folderIDs"`
		ProjectID        string            `yaml:"projectID,omitempty"`
		ProjectLabels    map[string]string `yaml:"projectLabels"`
		ProjectIDs       map[string]string `yaml:"projectIDs"`
		Stackdriver      struct {
			ProjectID  string            `yaml:"projectID,omitempty"`
			ProjectIDs map[string]string `yaml:"projectIDs"`
		}
		Repository struct {
			Name string `valid:"isNotZeroValue"`
		}
		GAE struct {
			Region string `valid:"isNotZeroValue"`
		}
		GCB struct {
			QueueTTL string `yaml:"queueTtl"`
		}
		GCF struct {
			Region string `valid:"isNotZeroValue"`
		}
		GCS struct {
			Buckets struct {
				CAIExport struct {
					Name            string `yaml:",omitempty"`
					Names           map[string]string
					DeleteAgeInDays int64 `yaml:"deleteAgeInDays,omitempty"`
				} `yaml:"CAIExport"`
				AssetsJSONFile struct {
					Name            string `yaml:",omitempty"`
					Names           map[string]string
					DeleteAgeInDays int64 `yaml:"deleteAgeInDays,omitempty"`
				} `yaml:"assetsJSONFile"`
			}
		}
		Bigquery struct {
			Dataset struct {
				Name     string `valid:"isNotZeroValue"`
				Location string `valid:"isNotZeroValue"`
			}
			Views struct {
				IntervalDays int64 `yaml:"intervalDays,omitempty"`
			}
		}
		Pubsub struct {
			TopicNames struct {
				IAMPolicies         string `yaml:"IAMPolicies" valid:"isNotZeroValue"`
				RAMViolation        string `yaml:"RAMViolation" valid:"isNotZeroValue"`
				RAMComplianceStatus string `yaml:"RAMComplianceStatus" valid:"isNotZeroValue"`
				GCIGroupMembers     string `yaml:"GCIGroupMembers"`
				GCIGroupSettings    string `yaml:"GCIGroupSettings"`
			} `yaml:"topicNames"`
		}
		FireStore struct {
			CollectionIDs struct {
				Assets string `valid:"isNotZeroValue"`
			} `yaml:"collectionIDs"`
		}
	}
	Monitoring struct {
		OrganizationIDs []string `yaml:"organizationIDs"`
		LabelKeyNames   struct {
			Owner             string `valid:"isNotZeroValue"`
			ViolationResolver string `yaml:"violationResolver" valid:"isNotZeroValue"`
		} `yaml:"labelKeyNames"`
		DefaultSchedulers map[string]struct {
			JobName  string `yaml:"jobName"`
			Schedule string
		} `yaml:"defaultSchedulers"`
		DirectoryCustomerIDs map[string]struct {
			SuperAdminEmail string `yaml:"superAdminEmail"`
		} `yaml:"directoryCustomerIDs"`
		ListGroupsDefaultSchedulers map[string]struct {
			JobName  string `yaml:"jobName"`
			Schedule string
		} `yaml:"listGroupsDefaultSchedulers"`
		AssetTypes struct {
			IAMPolicies []string `yaml:"iamPolicies"`
			Resources   []string `yaml:"resources"`
		} `yaml:"assetTypes"`
	}
}
