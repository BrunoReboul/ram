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

package ram

import (
	"fmt"
	"io/ioutil"

	"github.com/BrunoReboul/ram/utilities/validater"
	"gopkg.in/yaml.v2"
)

// Settings file names
const (
	SolutionSettingsFileName     = "solution.yaml"
	ServiceSettingsFileName      = "service.yaml"
	InstanceSettingsFileName     = "instance.yaml"
	MicroserviceParentFolderName = "services"
	InstancesFolderName          = "instances"
)

// Service structure
type Service struct {
	ConfigFilePath  string
	ServiceSettings Configurer
	Instances       map[string]Instance
}

// Instance structure
type Instance struct {
	ConfigFilePath   string
	InstanceSettings Configurer
}

// Configurer interface
type Configurer interface {
	ReadConfigFile(path string) (err error)
	Validate() (err error)
	Situate(situation interface{}) (err error)
	ReadValidateSituate(path string, situation interface{}) (err error)
}

// InstanceSituation settings used to situate an instance
type InstanceSituation struct {
	Solution     *SolutionSettings
	Service      Configurer
	InstanceName string
}

// SolutionSettings settings common to all services / all instances
type SolutionSettings struct {
	Hosting struct {
		BillingAccountID string            `yaml:"billingAccountID"`
		FolderID         string            `yaml:",omitempty"`
		FolderIDs        map[string]string `yaml:"folderIDs"`
		ProjectID        string            `yaml:",omitempty"`
		ProjectIDs       map[string]string `yaml:"projectIDs"`
		Stackdriver      struct {
			ProjectID  string            `yaml:",omitempty"`
			ProjectIDs map[string]string `yaml:"projectIDs"`
		}
		Repository struct {
			Name string `valid:"isNotZeroValue"`
		}
		GAE struct {
			Region string `valid:"isNotZeroValue"`
		}
		GCF struct {
			Region string `valid:"isNotZeroValue"`
		}
		GCS struct {
			Buckets struct {
				CAIExport struct {
					Name  string `yaml:",omitempty"`
					Names map[string]string
				} `yaml:"CAIExport"`
				AssetsJSONFile struct {
					Name  string `yaml:",omitempty"`
					Names map[string]string
				} `yaml:"assetsJSONFile"`
			}
		}
		Bigquery struct {
			Dataset struct {
				Name     string `valid:"isNotZeroValue"`
				Location string `valid:"isNotZeroValue"`
			}
		}
		Pubsub struct {
			TopicNames struct {
				IAMPolicies         string `yaml:"IAMPolicies" valid:"isNotZeroValue"`
				RAMViolation        string `yaml:"RAMViolation" valid:"isNotZeroValue"`
				RAMComplianceStatus string `yaml:"RAMComplianceStatus" valid:"isNotZeroValue"`
			} `yaml:"topicNames"`
		}
		FireStore struct {
			CollectionIDs struct {
				Assets string `valid:"isNotZeroValue"`
			} `yaml:"collectionIDs"`
		}
	}
	Monitoring struct {
		OrganizationIDs      []string          `yaml:"organizationIDs"`
		DirectoryCustomerIDs map[string]string `yaml:"directoryCustomerIDs"`
		LabelKeyNames        struct {
			Owner             string `valid:"isNotZeroValue"`
			ViolationResolver string `yaml:"violationResolver" valid:"isNotZeroValue"`
		} `yaml:"labelKeyNames"`
		AssetTypes struct {
			IAMPolicies []string `yaml:"iamPolicies"`
			Resources   []string `yaml:"resources"`
		} `yaml:"assetTypes"`
	}
}

// ReadUnmarshalYAML Read bytes from a given path and unmarshal assuming YAML format
func ReadUnmarshalYAML(path string, settings interface{}) (err error) {
	settingsYAML, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(settingsYAML, settings)
	if err != nil {
		return err
	}
	return nil
}

// ReadConfigFile reads and validates solution settings from a config file
func (settings *SolutionSettings) ReadConfigFile(path string) (err error) {
	return ReadUnmarshalYAML(path, settings)
}

// Validate validates the settings
func (settings *SolutionSettings) Validate() (err error) {
	err = validater.ValidateStruct(settings, "solutionSettings")
	if err != nil {
		return err
	}
	return nil
}

// Situate set settings from settings based on a given situation
// Situation is the environment name (string)
// Set settings are: folderID, projectID, Stackdriver projectID, Buckets names
func (settings *SolutionSettings) Situate(situation interface{}) (err error) {
	if environmentName, ok := situation.(string); ok {
		settings.Hosting.FolderID = settings.Hosting.FolderIDs[environmentName]
		settings.Hosting.ProjectID = settings.Hosting.ProjectIDs[environmentName]
		settings.Hosting.Stackdriver.ProjectID = settings.Hosting.Stackdriver.ProjectIDs[environmentName]
		settings.Hosting.GCS.Buckets.CAIExport.Name = settings.Hosting.GCS.Buckets.CAIExport.Names[environmentName]
		settings.Hosting.GCS.Buckets.AssetsJSONFile.Name = settings.Hosting.GCS.Buckets.AssetsJSONFile.Names[environmentName]
		return nil
	}
	return fmt.Errorf("situation is expected to be the environment name as a string")
}

// ReadValidateSituate reads settings from a config file, validates then, situates them
func (settings *SolutionSettings) ReadValidateSituate(path string, situation interface{}) (err error) {
	err = settings.ReadConfigFile(path)
	if err != nil {
		return err
	}
	err = settings.Validate()
	if err != nil {
		return err
	}
	err = settings.Situate(situation)
	if err != nil {
		return err
	}
	return nil
}

// NewSolutionSettings create a solution settings structure
func NewSolutionSettings() *SolutionSettings {
	return &SolutionSettings{}
}
