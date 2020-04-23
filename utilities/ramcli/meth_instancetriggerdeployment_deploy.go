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

package ramcli

import (
	"fmt"

	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/grm"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iam"
	"github.com/BrunoReboul/ram/utilities/ram"
)

// Deploy an instance trigger
func (instanceTriggerDeployment *InstanceTriggerDeployment) Deploy() (err error) {
	if err = instanceTriggerDeployment.deployGSUAPI(); err != nil {
		return err
	}
	if err = instanceTriggerDeployment.deployGRMBindings(); err != nil {
		return err
	}
	if err = instanceTriggerDeployment.deployIAMServiceAccount(); err != nil {
		return err
	}
	if err = instanceTriggerDeployment.deployIAMBindings(); err != nil {
		return err
	}
	if err = instanceTriggerDeployment.deployGCBTrigger(); err != nil {
		return err
	}
	return nil
}

func (instanceTriggerDeployment *InstanceTriggerDeployment) deployGSUAPI() (err error) {
	var deployment ram.Deployment
	apiDeployment := gsu.NewAPIDeployment()
	apiDeployment.Core = instanceTriggerDeployment.Core
	apiDeployment.Settings.Service.GSU = instanceTriggerDeployment.Settings.Service.GSU
	deployment = apiDeployment
	return deployment.Deploy()
}

func (instanceTriggerDeployment *InstanceTriggerDeployment) deployGRMBindings() (err error) {
	var deployment ram.Deployment
	bindingsDeployment := grm.NewBindingsDeployment()
	bindingsDeployment.Core = instanceTriggerDeployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", instanceTriggerDeployment.Core.ProjectNumber)
	bindingsDeployment.Settings.Service.GRM = instanceTriggerDeployment.Settings.Service.GCB.ServiceAccountBindings.ResourceManager
	deployment = bindingsDeployment
	return deployment.Deploy()
}

func (instanceTriggerDeployment *InstanceTriggerDeployment) deployIAMServiceAccount() (err error) {
	var deployment ram.Deployment
	serviceAccountDeployment := iam.NewServiceaccountDeployment()
	serviceAccountDeployment.Core = instanceTriggerDeployment.Core
	deployment = serviceAccountDeployment
	return deployment.Deploy()
}

func (instanceTriggerDeployment *InstanceTriggerDeployment) deployIAMBindings() (err error) {
	var deployment ram.Deployment
	bindingsDeployment := iam.NewBindingsDeployment()
	bindingsDeployment.Core = instanceTriggerDeployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", instanceTriggerDeployment.Core.ProjectNumber)
	bindingsDeployment.Settings.Service.IAM = instanceTriggerDeployment.Settings.Service.GCB.ServiceAccountBindings.IAM
	deployment = bindingsDeployment
	return deployment.Deploy()
}

func (instanceTriggerDeployment *InstanceTriggerDeployment) deployGCBTrigger() (err error) {
	var deployment ram.Deployment
	triggerDeployment := gcb.NewTriggerDeployment()
	triggerDeployment.Core = instanceTriggerDeployment.Core
	triggerDeployment.Settings.Service.GCB = instanceTriggerDeployment.Settings.Service.GCB
	deployment = triggerDeployment
	return deployment.Deploy()
}
