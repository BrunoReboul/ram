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

	"github.com/BrunoReboul/ram/utilities/bil"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/grm"
	"github.com/BrunoReboul/ram/utilities/gsr"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/iam"
)

// deployInstanceTrigger an instance trigger
func (deployment *Deployment) deployInstanceTrigger() (err error) {
	if err = deployment.deployGRMFolder(); err != nil {
		return err
	}
	if err = deployment.deployGRMProject(); err != nil {
		return err
	}
	if err = deployment.enableBILBillingAccountOnProject(); err != nil {
		return err
	}
	if err = deployment.deployGSUAPI(); err != nil {
		return err
	}
	if err = deployment.deployGRMBindings(); err != nil {
		return err
	}
	if err = deployment.deployIAMServiceAccount(); err != nil {
		return err
	}
	if err = deployment.deployIAMBindings(); err != nil {
		return err
	}
	if err = deployment.deployGSRRepo(); err != nil {
		return err
	}
	if err = deployment.deployGCBTrigger(); err != nil {
		return err
	}
	return nil
}

func (deployment *Deployment) deployGRMFolder() (err error) {
	folderDeployment := grm.NewFolderDeployment()
	folderDeployment.Core = &deployment.Core
	return folderDeployment.Deploy()
}

func (deployment *Deployment) deployGRMProject() (err error) {
	projectDeployment := grm.NewProjectDeployment()
	projectDeployment.Core = &deployment.Core
	return projectDeployment.Deploy()
}

func (deployment *Deployment) enableBILBillingAccountOnProject() (err error) {
	projectBillingAccount := bil.NewProjectBillingAccount()
	projectBillingAccount.Core = &deployment.Core
	return projectBillingAccount.Enable()
}

func (deployment *Deployment) deployGSUAPI() (err error) {
	apiDeployment := gsu.NewAPIDeployment()
	apiDeployment.Core = &deployment.Core
	apiDeployment.Settings.Service.GSU = deployment.Settings.Service.GSU
	return apiDeployment.Deploy()
}

func (deployment *Deployment) deployGRMBindings() (err error) {
	bindingsDeployment := grm.NewBindingsDeployment()
	bindingsDeployment.Core = &deployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
	bindingsDeployment.Settings.Service.GRM = deployment.Settings.Service.GCB.ServiceAccountBindings.ResourceManager
	return bindingsDeployment.Deploy()
}

func (deployment *Deployment) deployIAMServiceAccount() (err error) {
	serviceAccountDeployment := iam.NewServiceaccountDeployment()
	serviceAccountDeployment.Core = &deployment.Core
	return serviceAccountDeployment.Deploy()
}

func (deployment *Deployment) deployIAMBindings() (err error) {
	bindingsDeployment := iam.NewBindingsDeployment()
	bindingsDeployment.Core = &deployment.Core
	bindingsDeployment.Artifacts.Member = fmt.Sprintf("serviceAccount:%d@cloudbuild.gserviceaccount.com", deployment.Core.ProjectNumber)
	bindingsDeployment.Settings.Service.IAM = deployment.Settings.Service.GCB.ServiceAccountBindings.IAM
	return bindingsDeployment.Deploy()
}

func (deployment *Deployment) deployGSRRepo() (err error) {
	repoDeployment := gsr.NewRepoDeployment()
	repoDeployment.Core = &deployment.Core
	return repoDeployment.Deploy()
}

func (deployment *Deployment) deployGCBTrigger() (err error) {
	triggerDeployment := gcb.NewTriggerDeployment()
	triggerDeployment.Core = &deployment.Core
	triggerDeployment.Settings.Service.GCB = deployment.Settings.Service.GCB
	return triggerDeployment.Deploy()
}
