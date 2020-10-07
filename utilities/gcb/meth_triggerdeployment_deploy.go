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
	"reflect"
	"strings"

	"github.com/BrunoReboul/ram/utilities/erm"
	"github.com/BrunoReboul/ram/utilities/solution"
	"google.golang.org/api/cloudbuild/v1"
)

const retries = 5

var globalTriggerDeployment *TriggerDeployment
var count int

// Permission cloudbuild.builds.get is required in complemenet of cloudbuild.builds.list, event if 'get' API is not used

// Deploy delete if exist, then create a cloud build trigger to deploy a microservice instance
func (triggerDeployment *TriggerDeployment) Deploy() (err error) {
	// log.Printf("%s gcb cloud build trigger", triggerDeployment.Core.InstanceName)
	if triggerDeployment.Settings.Service.GCB.QueueTTL == "" {
		triggerDeployment.Settings.Service.GCB.QueueTTL = triggerDeployment.Core.SolutionSettings.Hosting.GCB.QueueTTL
	}
	triggerDeployment.Artifacts.ProjectsTriggersService = triggerDeployment.Core.Services.CloudbuildService.Projects.Triggers
	triggerDeployment.situate()
	// ffo.JSONMarshalIndentPrint(&triggerDeployment.Artifacts.BuildTrigger)
	globalTriggerDeployment = triggerDeployment
	if triggerDeployment.Core.Commands.Check {
		if err = triggerDeployment.checkTrigger(); err != nil {
			return err
		}
		log.Printf("%s gcb trigger checked", globalTriggerDeployment.Core.InstanceName)
	} else {
		if err = triggerDeployment.deleteTriggers(); err != nil {
			return err
		}
		for i := 0; i < retries; i++ {
			buildTrigger, err := triggerDeployment.Artifacts.ProjectsTriggersService.Create(triggerDeployment.Core.SolutionSettings.Hosting.ProjectID,
				&triggerDeployment.Artifacts.BuildTrigger).Context(triggerDeployment.Core.Ctx).Do()
			if err != nil {
				if erm.IsNotTransientElseWait(err, 5) {
					return err
				}
			} else {
				// ffo.JSONMarshalIndentPrint(buildTrigger)
				log.Printf("%s gcb created trigger %s id %s with tag filter %s", globalTriggerDeployment.Core.InstanceName, buildTrigger.Name, buildTrigger.Id, buildTrigger.TriggerTemplate.TagName)
				break
			}
		}
	}
	return nil
}

func (triggerDeployment *TriggerDeployment) checkTrigger() (err error) {
	count = 0
	if err = triggerDeployment.Artifacts.ProjectsTriggersService.List(triggerDeployment.Core.SolutionSettings.Hosting.ProjectID).Pages(triggerDeployment.Core.Ctx, browseTriggerToCheck); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s gcb trigger NOT found for this instance", globalTriggerDeployment.Core.InstanceName)
	}
	return nil
}

func browseTriggerToCheck(response *cloudbuild.ListBuildTriggersResponse) error {
	for _, buildtrigger := range response.Triggers {
		if buildtrigger.Name == globalTriggerDeployment.Artifacts.BuildTrigger.Name {
			count++
			if err := checkBuildTrigger(buildtrigger); err != nil {
				return err
			}
		}
	}
	if count > 1 {
		return fmt.Errorf("%s gcb found more than one trigger for this instance", globalTriggerDeployment.Core.InstanceName)
	}
	return nil
}

func checkBuildTrigger(buildtrigger *cloudbuild.BuildTrigger) (err error) {
	// A least one trigger maching the instance name has been found, check its configuration
	var s string
	if buildtrigger.Description != globalTriggerDeployment.Artifacts.BuildTrigger.Description {
		s = fmt.Sprintf("%sdescription\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.Description,
			buildtrigger.Description)
	}
	if len(buildtrigger.Build.Steps) != len(globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps) {
		s = fmt.Sprintf("%sunexpected_number_of_steps\nwant %d\nhave %d\n", s,
			len(globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps),
			len(buildtrigger.Build.Steps))
	} else {
		for i := range globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps {
			if globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Id != buildtrigger.Build.Steps[i].Id {
				s = fmt.Sprintf("%sbuild_step_%d_id\nwant %s\nhave %s\n", s, i,
					globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Id,
					buildtrigger.Build.Steps[i].Id)
			}
			if globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Name != buildtrigger.Build.Steps[i].Name {
				s = fmt.Sprintf("%sbuild_step_%d_name\nwant %s\nhave %s\n", s, i,
					globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Name,
					buildtrigger.Build.Steps[i].Name)
			}
			if !reflect.DeepEqual(globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Args, buildtrigger.Build.Steps[i].Args) {
				s = fmt.Sprintf("%sbuild_step_%d_args\nwant %s\nhave %s\n", s, i,
					strings.Join(globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Args[:], " "),
					strings.Join(buildtrigger.Build.Steps[i].Args[:], " "))
			}
			if globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Entrypoint != buildtrigger.Build.Steps[i].Entrypoint {
				s = fmt.Sprintf("%sbuild_step_%d_entryPoint\nwant %s\nhave %s\n", s, i,
					globalTriggerDeployment.Artifacts.BuildTrigger.Build.Steps[i].Entrypoint,
					buildtrigger.Build.Steps[i].Entrypoint)
			}
		}
	}
	if buildtrigger.Build.Timeout != globalTriggerDeployment.Artifacts.BuildTrigger.Build.Timeout {
		s = fmt.Sprintf("%sbuild_timeout\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.Build.Timeout,
			buildtrigger.Build.Timeout)
	}
	if buildtrigger.Build.QueueTtl != globalTriggerDeployment.Artifacts.BuildTrigger.Build.QueueTtl {
		s = fmt.Sprintf("%sbuild_queueTtl\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.Build.QueueTtl,
			buildtrigger.Build.QueueTtl)
	}
	if !reflect.DeepEqual(buildtrigger.Build.Tags, globalTriggerDeployment.Artifacts.BuildTrigger.Build.Tags) {
		s = fmt.Sprintf("%sbuild_tags\nwant %s\nhave %s\n", s,
			strings.Join(globalTriggerDeployment.Artifacts.BuildTrigger.Build.Tags[:], ","),
			strings.Join(buildtrigger.Build.Tags[:], ","))
	}
	if buildtrigger.TriggerTemplate.ProjectId != globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.ProjectId {
		s = fmt.Sprintf("%striggerTemplate_projectId\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.ProjectId,
			buildtrigger.TriggerTemplate.ProjectId)
	}
	if buildtrigger.TriggerTemplate.RepoName != globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.RepoName {
		s = fmt.Sprintf("%striggerTemplate_repoName\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.RepoName,
			buildtrigger.TriggerTemplate.RepoName)
	}
	if buildtrigger.TriggerTemplate.TagName != globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.TagName {
		s = fmt.Sprintf("%striggerTemplate_tagName\nwant %s\nhave %s\n", s,
			globalTriggerDeployment.Artifacts.BuildTrigger.TriggerTemplate.TagName, buildtrigger.TriggerTemplate.TagName)
	}

	if len(s) > 0 {
		return fmt.Errorf("%s gcb invalid trigger configuration:\n%s", globalTriggerDeployment.Core.InstanceName, s)
	}
	return nil
}

func (triggerDeployment *TriggerDeployment) deleteTriggers() (err error) {
	err = triggerDeployment.Artifacts.ProjectsTriggersService.List(triggerDeployment.Core.SolutionSettings.Hosting.ProjectID).Pages(triggerDeployment.Core.Ctx, browseTriggerToDelete)
	if err != nil {
		return fmt.Errorf("ProjectsTriggersService.List for deleting %v", err)
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
	triggerDeployment.Artifacts.BuildTrigger.Name = strings.Replace(triggerDeployment.Core.InstanceName, "_", "-", -1)
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
	build.QueueTtl = triggerDeployment.Settings.Service.GCB.QueueTTL
	build.Tags = []string{triggerDeployment.Core.ServiceName,
		triggerDeployment.Core.InstanceName,
		solution.SolutionName}
	return &build
}

func (triggerDeployment *TriggerDeployment) tagRegex() (tagRegex string) {
	instanceTagRegex := fmt.Sprintf("(^%s-v\\d*.\\d*.\\d*-%s)", triggerDeployment.Core.InstanceName, triggerDeployment.Core.EnvironmentName)
	serviceTagRegex := fmt.Sprintf("(^%s-v\\d*.\\d*.\\d*-%s)", triggerDeployment.Core.ServiceName, triggerDeployment.Core.EnvironmentName)
	solutionTagRegex := fmt.Sprintf("(^%s-v\\d*.\\d*.\\d*-%s)", solution.SolutionName, triggerDeployment.Core.EnvironmentName)
	if triggerDeployment.Artifacts.AssetShortTypeName == "" {
		return fmt.Sprintf("%s|%s|%s", instanceTagRegex, serviceTagRegex, solutionTagRegex)
	}
	assetTypeRegex := fmt.Sprintf("(^%s-v\\d*.\\d*.\\d*-%s)", triggerDeployment.Artifacts.AssetShortTypeName, triggerDeployment.Core.EnvironmentName)
	return fmt.Sprintf("%s|%s|%s|%s", instanceTagRegex, serviceTagRegex, solutionTagRegex, assetTypeRegex)

}
