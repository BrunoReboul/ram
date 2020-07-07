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

package grm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// Deploy ProjectDeployment check if a project exist, try to create it is missing and role projectCreator as been delegated (optional)
func (projectDeployment *ProjectDeployment) Deploy() (err error) {
	log.Printf("%s grm resource manager project", projectDeployment.Core.InstanceName)
	projectsService := projectDeployment.Core.Services.CloudresourcemanagerService.Projects
	operationsService := projectDeployment.Core.Services.CloudresourcemanagerService.Operations
	project, err := projectsService.Get(projectDeployment.Core.SolutionSettings.Hosting.ProjectID).Context(projectDeployment.Core.Ctx).Do()
	if err != nil {
		// When a project is not found the API returns 403 forbiden instead of 404 not found
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "403") {
			var parent cloudresourcemanager.ResourceId
			parent.Type = "folder"
			parent.Id = projectDeployment.Core.SolutionSettings.Hosting.FolderID
			var projectToCreate cloudresourcemanager.Project
			projectToCreate.ProjectId = projectDeployment.Core.SolutionSettings.Hosting.ProjectID
			projectToCreate.Name = projectDeployment.Core.SolutionSettings.Hosting.ProjectID
			projectToCreate.Parent = &parent
			projectToCreate.Labels = projectDeployment.Core.SolutionSettings.Hosting.ProjectLabels
			operation, err := projectsService.Create(&projectToCreate).Context(projectDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "403") {
					log.Printf("%s grm WARNING impossible to CREATE project %v", projectDeployment.Core.InstanceName, err)
					return nil
				}
				return fmt.Errorf("grm projectsService.Create(&projectToCreate) %v", err)
			}
			operationName := operation.Name
			log.Printf("%s grm create project %s operation started", projectDeployment.Core.InstanceName, projectDeployment.Core.SolutionSettings.Hosting.ProjectID)
			log.Println(operationName)
			for {
				time.Sleep(5 * time.Second)
				for i := 0; i < Retries; i++ {
					operation, err = operationsService.Get(operationName).Context(projectDeployment.Core.Ctx).Do()
					if err != nil {
						if strings.Contains(err.Error(), "500") && strings.Contains(err.Error(), "backendError") {
							log.Printf("%s ERROR getting operation status, iteration %d, wait 5 sec and retry %v", projectDeployment.Core.InstanceName, i, err)
							time.Sleep(5 * time.Second)
						} else {
							return err
						}
					}
				}
				if err != nil {
					return err
				}
				if operation.Done {
					break
				}
			}
			ffo.JSONMarshalIndentPrint(operation)
			log.Printf("%s grm project %s created", projectDeployment.Core.InstanceName, projectDeployment.Core.SolutionSettings.Hosting.ProjectID)
		} else {
			return err
		}
	} else {
		if project.LifecycleState != "ACTIVE" {
			return fmt.Errorf("%s grm project %s %s %d is in state %s while it should be ACTIVE", projectDeployment.Core.InstanceName,
				project.ProjectId,
				project.Name,
				project.ProjectNumber,
				project.LifecycleState)
		}
		log.Printf("%s grm project found %s %s %d parent %s %s", projectDeployment.Core.InstanceName,
			project.ProjectId,
			project.Name,
			project.ProjectNumber,
			project.Parent.Type,
			project.Parent.Id)
	}
	return nil
}
