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

package publish2fs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/ram"
	"github.com/google/uuid"
	"google.golang.org/api/cloudfunctions/v1"
	"gopkg.in/yaml.v2"
)

const (
	description  = "publish %s assets resource feeds as FireStore documents in collection %s"
	eventService = "pubsub.googleapis.com"
	eventType    = "google.pubsub.topic.publish"
	gcfType      = "backgroundPubSub"
	serviceName  = "publish2fs"
)

// ServiceSettings defines service settings common to all service instances
type ServiceSettings struct {
	GCF struct {
		AvailableMemoryMb   int64  `yaml:"availableMemoryMb" valid:"isAvailableMemory"`
		RetryTimeOutSeconds int64  `yaml:"retryTimeOutSeconds"`
		Timeout             string `yaml:"timeout"`
	}
}

// InstanceSettings instance specific settings
type InstanceSettings struct {
	GCF struct {
		TriggerTopic string `yaml:"triggerTopic"`
	}
}

// Settings flat settings structure: solution - service - instance
type Settings struct {
	Solution ram.SolutionSettings
	Service  ServiceSettings
	Instance InstanceSettings
}

// GoGCFDeployment settings and Artifacts structure
type GoGCFDeployment struct {
	DumpTimestamp time.Time
	Settings      Settings
	Artifacts     gcf.GoGCFArtifacts
}

// NewGoGCFDeployment create a service settings structure
func NewGoGCFDeployment() *GoGCFDeployment {
	return &GoGCFDeployment{}
}

// DeployGoCloudFunction deploy an instance of a microservice as a Go cloud function
func (goGCFDeployment *GoGCFDeployment) DeployGoCloudFunction() (err error) {
	start := time.Now()
	log.Printf("%s deploy cloud function %s", goGCFDeployment.Artifacts.InstanceName, serviceName)
	err = goGCFDeployment.readValidate()
	if err != nil {
		return err
	}
	err = goGCFDeployment.situate()
	if err != nil {
		return err
	}
	log.Printf("%s settings read, validated, situated", goGCFDeployment.Artifacts.InstanceName)
	err = goGCFDeployment.Artifacts.CreateGCFServiceAccount()
	if err != nil {
		return err
	}
	log.Printf("%s service account found or created", goGCFDeployment.Artifacts.InstanceName)
	err = ram.ZipSource(goGCFDeployment.Artifacts.CloudFunctionZipFullPath, goGCFDeployment.Artifacts.ZipFiles)
	if err != nil {
		return err
	}
	log.Printf("%s sources zipped", goGCFDeployment.Artifacts.InstanceName)
	err = goGCFDeployment.Artifacts.GetUploadURL(goGCFDeployment.Settings.Solution.Hosting.ProjectID,
		goGCFDeployment.Settings.Solution.Hosting.GCF.Region)
	if err != nil {
		return err
	}
	log.Printf("%s signed URL for upload retreived", goGCFDeployment.Artifacts.InstanceName)
	response, err := goGCFDeployment.Artifacts.UploadZipUsingSignedURL()
	if err != nil {
		return err
	}
	log.Printf("%s upload %s response status code: %v", goGCFDeployment.Artifacts.InstanceName, goGCFDeployment.Artifacts.CloudFunctionZipFullPath, response.StatusCode)
	err = goGCFDeployment.Artifacts.CreatePatchCloudFunction()
	if err != nil {
		return err
	}
	err = os.Remove(goGCFDeployment.Artifacts.CloudFunctionZipFullPath)
	if err != nil {
		return err
	}
	log.Printf("%s remove file %s", goGCFDeployment.Artifacts.InstanceName, goGCFDeployment.Artifacts.CloudFunctionZipFullPath)
	if err != nil {
		return err
	}
	log.Printf("%s done in %v minutes", goGCFDeployment.Artifacts.InstanceName, time.Since(start).Minutes())
	return nil
}

func (goGCFDeployment *GoGCFDeployment) readValidate() (err error) {
	solutionConfigFilePath := fmt.Sprintf("%s/%s", goGCFDeployment.Artifacts.RepositoryPath, ram.SolutionSettingsFileName)
	err = ram.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &goGCFDeployment.Settings.Solution)
	if err != nil {
		return err
	}

	serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", goGCFDeployment.Artifacts.RepositoryPath, ram.MicroserviceParentFolderName, serviceName, ram.ServiceSettingsFileName)
	err = ram.ReadValidate(serviceName, "ServiceSettings", serviceConfigFilePath, &goGCFDeployment.Settings.Service)
	if err != nil {
		return err
	}

	instanceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", goGCFDeployment.Artifacts.RepositoryPath, ram.MicroserviceParentFolderName, serviceName, ram.InstancesFolderName, goGCFDeployment.Artifacts.InstanceName, ram.InstanceSettingsFileName)
	err = ram.ReadValidate(goGCFDeployment.Artifacts.InstanceName, "InstanceSettings", instanceConfigFilePath, &goGCFDeployment.Settings.Instance)
	if err != nil {
		return err
	}
	return nil
}

func (goGCFDeployment *GoGCFDeployment) situate() (err error) {
	goGCFDeployment.Settings.Solution.Situate(goGCFDeployment.Artifacts.EnvironmentName)

	goGCFDeployment.Artifacts.ServiceName = serviceName
	goGCFDeployment.Artifacts.ProjectID = goGCFDeployment.Settings.Solution.Hosting.ProjectID
	goGCFDeployment.Artifacts.Region = goGCFDeployment.Settings.Solution.Hosting.GCF.Region
	goGCFDeployment.Artifacts.CloudFunctionZipFullPath = fmt.Sprintf("./%s.zip", uuid.New())
	var failurePolicy cloudfunctions.FailurePolicy
	retry := cloudfunctions.Retry{}
	failurePolicy.Retry = &retry

	var eventTrigger cloudfunctions.EventTrigger
	eventTrigger.EventType = eventType
	eventTrigger.Resource = fmt.Sprintf("projects/%s/topics/%s", goGCFDeployment.Settings.Solution.Hosting.ProjectID, goGCFDeployment.Settings.Instance.GCF.TriggerTopic)
	eventTrigger.Service = eventService
	eventTrigger.FailurePolicy = &failurePolicy

	envVars := make(map[string]string)
	envVars["RETRYTIMEOUTSECONDS"] = strconv.FormatInt(goGCFDeployment.Settings.Service.GCF.RetryTimeOutSeconds, 10)
	envVars["COLLECTION_ID"] = goGCFDeployment.Settings.Solution.Hosting.FireStore.CollectionIDs.Assets

	runTime, err := gcf.GetRunTime(goGCFDeployment.Artifacts.GoVersion)
	if err != nil {
		return err
	}

	goGCFDeployment.Artifacts.CloudFunction.AvailableMemoryMb = goGCFDeployment.Settings.Service.GCF.AvailableMemoryMb
	goGCFDeployment.Artifacts.CloudFunction.Description = fmt.Sprintf(description, goGCFDeployment.Settings.Instance.GCF.TriggerTopic, goGCFDeployment.Settings.Solution.Hosting.FireStore.CollectionIDs.Assets)
	goGCFDeployment.Artifacts.CloudFunction.EntryPoint = "EntryPoint"
	goGCFDeployment.Artifacts.CloudFunction.EnvironmentVariables = envVars
	goGCFDeployment.Artifacts.CloudFunction.EventTrigger = &eventTrigger
	goGCFDeployment.Artifacts.CloudFunction.Labels = map[string]string{"name": strings.ToLower(goGCFDeployment.Artifacts.InstanceName)}
	goGCFDeployment.Artifacts.CloudFunction.Name = fmt.Sprintf("projects/%s/locations/%s/functions/%s", goGCFDeployment.Settings.Solution.Hosting.ProjectID, goGCFDeployment.Settings.Solution.Hosting.GCF.Region, goGCFDeployment.Artifacts.InstanceName)
	goGCFDeployment.Artifacts.CloudFunction.Runtime = runTime
	goGCFDeployment.Artifacts.CloudFunction.Timeout = goGCFDeployment.Settings.Service.GCF.Timeout
	goGCFDeployment.Artifacts.CloudFunction.IngressSettings = "ALLOW_ALL"

	goGCFDeployment.Artifacts.ZipFiles = make(map[string]string)
	functionGoContent, err := gcf.MakeFunctionGoContent(gcfType, serviceName)
	if err != nil {
		return err
	}
	goGCFDeployment.Artifacts.ZipFiles["function.go"] = functionGoContent
	goGCFDeployment.Artifacts.ZipFiles["go.mod"] = gcf.MakeGoModContent(goGCFDeployment.Artifacts.GoVersion, goGCFDeployment.Artifacts.RAMVersion)

	// Keep ram.SettingsFileName as the last element of the map (himself)
	goGCFDeployment.DumpTimestamp = time.Now()
	GoGCFDeploymentYAMLBytes, err := yaml.Marshal(goGCFDeployment)
	if err != nil {
		return err
	}
	goGCFDeployment.Artifacts.ZipFiles[ram.SettingsFileName] = string(GoGCFDeploymentYAMLBytes)
	if goGCFDeployment.Artifacts.Dump {
		err := ram.DumpToYAMLFile(goGCFDeployment, fmt.Sprintf("%s/%s", goGCFDeployment.Artifacts.RepositoryPath, ram.SettingsFileName))
		if err != nil {
			return err
		}
	}
	return nil
}
