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
	"strconv"

	"github.com/BrunoReboul/ram/utilities/gcftemplate"
	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/cloudfunctions/v1"
)

const (
	description            = "publish %s assets resource feeds as FireStore documents in collection %s"
	eventProviderNamespace = "cloud.pubsub"
	eventResourceType      = "topic.publish"
	eventService           = "pubsub.googleapis.com"
	gcfType                = "backgroundPubSub"
	serviceName            = "publish2fs"
)

// Deployment flatten instance / service / solution structure
type Deployment struct {
	Settings struct {
		Solution ram.SolutionSettings
		Service  ServiceSettings
		Instance InstanceSettings
	}
	Artifacts struct {
		CloudFunction     cloudfunctions.CloudFunction
		GoModContent      string
		FunctionGoContent string
	}
}

// InstanceSettings instance specific settings
type InstanceSettings struct {
	GCF struct {
		TriggerTopic string `yaml:"triggerTopic"`
	}
}

// ServiceSettings defines service settings common to all service instances
type ServiceSettings struct {
	GCF struct {
		AvailableMemoryMb   int64  `yaml:"availableMemoryMb" valid:"isAvailableMemory"`
		RetryTimeOutSeconds int64  `yaml:"retryTimeOutSeconds"`
		Timeout             string `yaml:"timeout"`
	}
}

// NewDeployment create a service settings structure
func NewDeployment() *Deployment {
	return &Deployment{}
}

// Deploy deploy an instance of a microservice
func (deployment *Deployment) Deploy(goVersion, ramVersion, repositoryPath, environmentName, instanceName string, dump bool) (err error) {
	err = deployment.readValidate(repositoryPath, instanceName)
	if err != nil {
		return err
	}
	err = deployment.situate(goVersion, ramVersion, repositoryPath, instanceName, environmentName, dump)
	if err != nil {
		return err
	}
	return nil
}

func (deployment *Deployment) readValidate(repositoryPath, instanceName string) (err error) {
	solutionConfigFilePath := fmt.Sprintf("%s/%s", repositoryPath, ram.SolutionSettingsFileName)
	err = ram.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &deployment.Settings.Solution)
	if err != nil {
		return err
	}

	serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", repositoryPath, ram.MicroserviceParentFolderName, serviceName, ram.ServiceSettingsFileName)
	err = ram.ReadValidate(serviceName, "ServiceSettings", serviceConfigFilePath, &deployment.Settings.Service)
	if err != nil {
		return err
	}

	instanceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", repositoryPath, ram.MicroserviceParentFolderName, serviceName, ram.InstancesFolderName, instanceName, ram.InstanceSettingsFileName)
	err = ram.ReadValidate(instanceName, "InstanceSettings", instanceConfigFilePath, &deployment.Settings.Instance)
	if err != nil {
		return err
	}
	return nil
}

func (deployment *Deployment) situate(goVersion, ramVersion, repositoryPath, instanceName, environmentName string, dump bool) (err error) {
	deployment.Settings.Solution.Situate(environmentName)

	var failurePolicy cloudfunctions.FailurePolicy
	retry := cloudfunctions.Retry{}
	failurePolicy.Retry = &retry

	var eventTrigger cloudfunctions.EventTrigger
	eventTrigger.EventType = fmt.Sprintf("providers/%s/eventTypes/%s", eventProviderNamespace, eventResourceType)
	eventTrigger.Resource = fmt.Sprintf("projects/%s/topics/%s", deployment.Settings.Solution.Hosting.ProjectID, deployment.Settings.Instance.GCF.TriggerTopic)
	eventTrigger.Service = eventService
	eventTrigger.FailurePolicy = &failurePolicy

	envVars := make(map[string]string)
	envVars["RETRYTIMEOUTSECONDS"] = strconv.FormatInt(deployment.Settings.Service.GCF.RetryTimeOutSeconds, 10)
	envVars["COLLECTION_ID"] = deployment.Settings.Solution.Hosting.FireStore.CollectionIDs.Assets

	runTime, err := gcftemplate.GetRunTime(goVersion)
	if err != nil {
		return err
	}

	deployment.Artifacts.CloudFunction.AvailableMemoryMb = deployment.Settings.Service.GCF.AvailableMemoryMb
	deployment.Artifacts.CloudFunction.Description = fmt.Sprintf(description, deployment.Settings.Instance.GCF.TriggerTopic, deployment.Settings.Solution.Hosting.FireStore.CollectionIDs.Assets)
	deployment.Artifacts.CloudFunction.EntryPoint = "EntryPoint"
	deployment.Artifacts.CloudFunction.EnvironmentVariables = envVars
	deployment.Artifacts.CloudFunction.EventTrigger = &eventTrigger
	deployment.Artifacts.CloudFunction.Labels = map[string]string{"name": instanceName}
	deployment.Artifacts.CloudFunction.Name = fmt.Sprintf("projects/%s/locations/%s/functions/%s", deployment.Settings.Solution.Hosting.ProjectID, deployment.Settings.Solution.Hosting.GCF.Region, instanceName)
	deployment.Artifacts.CloudFunction.Runtime = runTime
	deployment.Artifacts.CloudFunction.ServiceAccountEmail = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceName, deployment.Settings.Solution.Hosting.ProjectID)
	deployment.Artifacts.CloudFunction.Timeout = deployment.Settings.Service.GCF.Timeout

	functionGoContent, err := gcftemplate.MakeFunctionGoContent(gcfType, serviceName)
	if err != nil {
		return err
	}
	deployment.Artifacts.GoModContent = gcftemplate.MakeGoModContent(goVersion, ramVersion)
	deployment.Artifacts.FunctionGoContent = functionGoContent

	if dump {
		err := ram.DumpToYAMLFile(deployment, fmt.Sprintf("%s/%s", repositoryPath, ram.DumpSettingsFileName))
		if err != nil {
			return err
		}
	}
	return nil
}
