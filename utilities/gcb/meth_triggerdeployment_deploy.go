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

package gcb

import (
	"fmt"
	"log"

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/cloudbuild/v1"
)

var globalTriggerDeployment *TriggerDeployment

// Deploy create or update a cloud build trigger to deploy a microservice instance
func (triggerDeployment *TriggerDeployment) Deploy() (err error) {
	triggerDeployment.situate()
	// ram.JSONMarshalIndentPrint(&triggerDeployment.Artifacts.BuildTrigger)
	globalTriggerDeployment = triggerDeployment
	triggerDeployment.deleteTriggers()
	buildTrigger, err := triggerDeployment.Artifacts.ProjectsTriggersService.Create(triggerDeployment.Core.SolutionSettings.Hosting.ProjectID,
		&triggerDeployment.Artifacts.BuildTrigger).Context(triggerDeployment.Core.Ctx).Do()
	if err != nil {
		return err
	}
	// ram.JSONMarshalIndentPrint(buildTrigger)
	log.Printf("%s gcb created trigger %s id %s with tag filter %s", globalTriggerDeployment.Core.InstanceName, buildTrigger.Name, buildTrigger.Id, buildTrigger.TriggerTemplate.TagName)
	return nil
}

func (triggerDeployment *TriggerDeployment) deleteTriggers() (err error) {
	err = triggerDeployment.Artifacts.ProjectsTriggersService.List(triggerDeployment.Core.SolutionSettings.Hosting.ProjectID).Pages(triggerDeployment.Core.Ctx, browseTriggerToDelete)
	if err != nil {
		return err
	}
	return nil
}

func browseTriggerToDelete(response *cloudbuild.ListBuildTriggersResponse) error {
	for _, buildtrigger := range response.Triggers {
		if buildtrigger.Name == globalTriggerDeployment.Artifacts.BuildTrigger.Name {
			deleteBuildTrigger(buildtrigger.Id)
		}
	}
	return nil
}

func deleteBuildTrigger(triggerID string) {
	_, err := globalTriggerDeployment.Artifacts.ProjectsTriggersService.Delete(globalTriggerDeployment.Core.SolutionSettings.Hosting.ProjectID,
		triggerID).Context(globalTriggerDeployment.Core.Ctx).Do()
	if err != nil {
		log.Printf("%s gcb ERROR when deleting existing trigger %s %s %v", globalTriggerDeployment.Core.InstanceName, triggerID, globalTriggerDeployment.Core.SolutionSettings.Hosting.ProjectID, err)
	} else {
		log.Printf("%s gcb deleted trigger id %s named %s", globalTriggerDeployment.Core.InstanceName, triggerID, globalTriggerDeployment.Artifacts.BuildTrigger.Name)
	}
}

func (triggerDeployment *TriggerDeployment) situate() {
	triggerDeployment.Artifacts.BuildTrigger.Name = fmt.Sprintf("%s-%s-cd",
		triggerDeployment.Core.EnvironmentName,
		triggerDeployment.Core.InstanceName)
	triggerDeployment.Artifacts.BuildTrigger.Description = fmt.Sprintf("Environment %s, Instance %s, Phase continuous deployment",
		triggerDeployment.Core.EnvironmentName,
		triggerDeployment.Core.InstanceName)
	triggerDeployment.Artifacts.BuildTrigger.Build = triggerDeployment.getInstanceDeploymentBuild()

	var repoSource cloudbuild.RepoSource
	repoSource.ProjectId = triggerDeployment.Core.SolutionSettings.Hosting.ProjectID
	repoSource.RepoName = triggerDeployment.Core.SolutionSettings.Hosting.Repository.Name
	repoSource.TagName = triggerDeployment.tagRegex()
	triggerDeployment.Artifacts.BuildTrigger.TriggerTemplate = &repoSource
}

func (triggerDeployment *TriggerDeployment) getInstanceDeploymentBuild() *cloudbuild.Build {
	var steps []*cloudbuild.BuildStep
	var step1, step2, step3 cloudbuild.BuildStep

	step1.Id = "build a fresh ram cli"
	step1.Name = "golang"
	step1.Args = []string{"go", "build", "ram.go"}
	steps = append(steps, &step1)

	step2.Id = "display ram executable info"
	step2.Name = "gcr.io/cloud-builders/gcloud"
	step2.Entrypoint = "bash"
	step2.Args = []string{"-c", "ls -al ram"}
	steps = append(steps, &step2)

	ramDeploymentCommand := fmt.Sprintf("./ram -deploy -environment=%s -service=%s -instance=%s",
		triggerDeployment.Core.EnvironmentName,
		triggerDeployment.Core.ServiceName,
		triggerDeployment.Core.InstanceName)
	step3.Id = fmt.Sprintf("deploy instance %s", triggerDeployment.Core.InstanceName)
	step3.Name = "gcr.io/cloud-builders/gcloud"
	step3.Entrypoint = "bash"
	step3.Args = []string{"-c", ramDeploymentCommand}
	steps = append(steps, &step3)

	var build cloudbuild.Build
	build.Steps = steps
	build.Timeout = triggerDeployment.Settings.Service.GCB.BuildTimeout
	build.Tags = []string{triggerDeployment.Core.ServiceName,
		triggerDeployment.Core.InstanceName,
		ram.SolutionName}
	return &build
}

func (triggerDeployment *TriggerDeployment) tagRegex() (tagRegex string) {
	instanceTagRegex := fmt.Sprintf("(%s-v\\d*.\\d*.\\d*-%s)", triggerDeployment.Core.InstanceName, triggerDeployment.Core.EnvironmentName)
	serviceTagRegex := fmt.Sprintf("(%s-v\\d*.\\d*.\\d*-%s)", triggerDeployment.Core.ServiceName, triggerDeployment.Core.EnvironmentName)
	solutionTagRegex := fmt.Sprintf("(%s-v\\d*.\\d*.\\d*-%s)", ram.SolutionName, triggerDeployment.Core.EnvironmentName)
	return fmt.Sprintf("%s|%s|%s", instanceTagRegex, serviceTagRegex, solutionTagRegex)
}
