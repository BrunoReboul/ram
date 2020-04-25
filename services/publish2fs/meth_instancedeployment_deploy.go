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
	"time"

	"github.com/BrunoReboul/ram/utilities/gcf"
	"gopkg.in/yaml.v2"

	"github.com/BrunoReboul/ram/utilities/grm"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iam"
	"github.com/BrunoReboul/ram/utilities/ram"
)

// Deploy a service instance
func (instanceDeployment *InstanceDeployment) Deploy() (err error) {
	start := time.Now()
	if err = instanceDeployment.readValidate(); err != nil {
		return err
	}
	instanceDeployment.situate()
	if err = instanceDeployment.deployGSUAPI(); err != nil {
		return err
	}
	if err = instanceDeployment.deployIAMServiceAccount(); err != nil {
		return err
	}
	if err = instanceDeployment.deployGRMBindings(); err != nil {
		return err
	}
	if err = instanceDeployment.deployIAMBindings(); err != nil {
		return err
	}
	if err = instanceDeployment.deployGCFFunction(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	log.Printf("%s done in %v minutes", instanceDeployment.Core.InstanceName, time.Since(start).Minutes())
	return nil
}

func (instanceDeployment *InstanceDeployment) readValidate() (err error) {
	serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", instanceDeployment.Core.RepositoryPath, ram.MicroserviceParentFolderName, instanceDeployment.Core.ServiceName, ram.ServiceSettingsFileName)
	err = ram.ReadValidate(instanceDeployment.Core.ServiceName, "ServiceSettings", serviceConfigFilePath, &instanceDeployment.Settings.Service)
	if err != nil {
		return err
	}
	instanceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", instanceDeployment.Core.RepositoryPath, ram.MicroserviceParentFolderName, instanceDeployment.Core.ServiceName, ram.InstancesFolderName, instanceDeployment.Core.InstanceName, ram.InstanceSettingsFileName)
	err = ram.ReadValidate(instanceDeployment.Core.InstanceName, "InstanceSettings", instanceConfigFilePath, &instanceDeployment.Settings.Instance)
	if err != nil {
		return err
	}
	return nil
}

func (instanceDeployment *InstanceDeployment) situate() {
	instanceDeployment.Settings.Service.GCF.FunctionType = "backgroundPubSub"
	instanceDeployment.Settings.Service.GCF.Description = fmt.Sprintf("publish %s assets resource feeds as FireStore documents in collection %s",
		instanceDeployment.Settings.Instance.GCF.TriggerTopic,
		instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets)
}

func (instanceDeployment *InstanceDeployment) deployGSUAPI() (err error) {
	var deployment ram.Deployment
	apiDeployment := gsu.NewAPIDeployment()
	apiDeployment.Core = instanceDeployment.Core
	apiDeployment.Settings.Service.GSU = instanceDeployment.Settings.Service.GSU
	deployment = apiDeployment
	return deployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployIAMServiceAccount() (err error) {
	var deployment ram.Deployment
	serviceAccountDeployment := iam.NewServiceaccountDeployment()
	serviceAccountDeployment.Core = instanceDeployment.Core
	deployment = serviceAccountDeployment
	return deployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployGRMBindings() (err error) {
	var deployment ram.Deployment
	bindingsDeployment := grm.NewBindingsDeployment()
	bindingsDeployment.Core = instanceDeployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", instanceDeployment.Core.ServiceName, instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)
	bindingsDeployment.Settings.Service.GRM = instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.ResourceManager
	deployment = bindingsDeployment
	return deployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployIAMBindings() (err error) {
	var deployment ram.Deployment
	bindingsDeployment := iam.NewBindingsDeployment()
	bindingsDeployment.Core = instanceDeployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", instanceDeployment.Core.ServiceName, instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)
	bindingsDeployment.Settings.Service.IAM = instanceDeployment.Settings.Service.GCF.ServiceAccountBindings.IAM
	deployment = bindingsDeployment
	return deployment.Deploy()
}

func (instanceDeployment *InstanceDeployment) deployGCFFunction() (err error) {
	instanceDeployment.DumpTimestamp = time.Now()
	instanceDeploymentYAMLBytes, err := yaml.Marshal(instanceDeployment)
	if err != nil {
		return err
	}
	var deployment ram.Deployment
	functionDeployment := gcf.NewFunctionDeployment()
	functionDeployment.Core = instanceDeployment.Core
	functionDeployment.Artifacts.InstanceDeploymentYAMLContent = string(instanceDeploymentYAMLBytes)
	functionDeployment.Settings.Service.GCF = instanceDeployment.Settings.Service.GCF
	functionDeployment.Settings.Instance.GCF = instanceDeployment.Settings.Instance.GCF
	deployment = functionDeployment
	return deployment.Deploy()
}
