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
	"context"
	"fmt"
	"log"

	"github.com/BrunoReboul/ram/utilities/deploy"

	"github.com/BrunoReboul/ram/utilities/ram"

	"github.com/BrunoReboul/ram/services/publish2fs"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
)

// Global structure for global variables
type Global struct {
	ctx                               context.Context
	cloudresourcemanagerService       *cloudresourcemanager.Service
	iamService                        *iam.Service
	operationsService                 *cloudfunctions.OperationsService
	projectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService
	projectsTriggersService           *cloudbuild.ProjectsTriggersService
	serviceusageService               *serviceusage.Service
	settings                          settings
}

// Settings is the full set of parameters
type settings struct {
	Commands struct {
		// Makeyaml     bool
		Maketrigger  bool
		Deploy       bool
		Dumpsettings bool
	}
	EnvironmentName             string
	InstanceFolderRelativePaths []string
	RepositoryPath              string
	Versions                    struct {
		Go  string
		RAM string
	}
}

// Initialize is to be executed in the init()
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	var err error
	var cloudfunctionsService *cloudfunctions.Service
	var cloudbuildService *cloudbuild.Service

	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("ERROR - google.FindDefaultCredentials %v", err)
	}

	cloudfunctionsService, err = cloudfunctions.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.operationsService = cloudfunctions.NewOperationsService(cloudfunctionsService)
	cloudfunctionsProjectsService := cloudfunctionsService.Projects
	global.projectsLocationsFunctionsService = cloudfunctionsProjectsService.Locations.Functions

	cloudbuildService, err = cloudbuild.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	cloudbuildProjectsService := cloudbuildService.Projects
	global.projectsTriggersService = cloudbuildProjectsService.Triggers

	global.iamService, err = iam.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.serviceusageService, err = serviceusage.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
}

// RAMCli Real-time Asset Monitor cli
func RAMCli(global *Global) (err error) {
	var deployment ram.Deployment
	global.settings.CheckArguments()

	var solutionSettings ram.SolutionSettings
	solutionConfigFilePath := fmt.Sprintf("%s/%s", global.settings.RepositoryPath, ram.SolutionSettingsFileName)
	err = ram.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &solutionSettings)
	if err != nil {
		log.Fatal(err)
	}
	solutionSettings.Situate(global.settings.EnvironmentName)

	var core deploy.Core
	core.Ctx = global.ctx
	core.EnvironmentName = global.settings.EnvironmentName
	core.RepositoryPath = global.settings.RepositoryPath
	core.RAMVersion = global.settings.Versions.RAM
	core.GoVersion = global.settings.Versions.Go
	core.Dump = global.settings.Commands.Dumpsettings
	log.Printf("found %d instance(s)", len(global.settings.InstanceFolderRelativePaths))
	core.ProjectNumber, err = getProjectNumber(global.ctx, global.cloudresourcemanagerService, solutionSettings.Hosting.ProjectID)

	if global.settings.Commands.Deploy {
		for _, instanceFolderRelativePath := range global.settings.InstanceFolderRelativePaths {
			serviceName, instanceName := GetServiceAndInstanceNames(instanceFolderRelativePath)
			switch serviceName {
			case "publish2fs":
				goGCFDeployment := publish2fs.NewGoGCFDeployment()

				goGCFDeployment.Artifacts.CloudresourcemanagerService = global.cloudresourcemanagerService
				goGCFDeployment.Artifacts.Ctx = global.ctx
				goGCFDeployment.Artifacts.Dump = global.settings.Commands.Dumpsettings
				goGCFDeployment.Artifacts.EnvironmentName = global.settings.EnvironmentName
				goGCFDeployment.Artifacts.GoVersion = global.settings.Versions.Go
				goGCFDeployment.Artifacts.IAMService = global.iamService
				goGCFDeployment.Artifacts.InstanceName = instanceName
				goGCFDeployment.Artifacts.OperationsService = global.operationsService
				goGCFDeployment.Artifacts.ProjectsLocationsFunctionsService = global.projectsLocationsFunctionsService
				goGCFDeployment.Artifacts.RAMVersion = global.settings.Versions.RAM
				goGCFDeployment.Artifacts.RepositoryPath = global.settings.RepositoryPath

				deployment = goGCFDeployment
				err := deployment.Deploy()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	if global.settings.Commands.Maketrigger {
		instanceTriggerDeployment := NewInstanceTrigger()
		instanceTriggerDeployment.Core = core
		instanceTriggerDeployment.Artifacts.ServiceusageService = global.serviceusageService
		instanceTriggerDeployment.Artifacts.ProjectsTriggersService = global.projectsTriggersService
		instanceTriggerDeployment.Artifacts.CloudresourcemanagerService = global.cloudresourcemanagerService

		instanceTriggerDeployment.Core.SolutionSettings = solutionSettings
		for _, instanceFolderRelativePath := range global.settings.InstanceFolderRelativePaths {
			serviceName, instanceName := GetServiceAndInstanceNames(instanceFolderRelativePath)
			instanceTriggerDeployment.Core.ServiceName = serviceName
			instanceTriggerDeployment.Core.InstanceName = instanceName

			serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", global.settings.RepositoryPath, ram.MicroserviceParentFolderName, serviceName, ram.ServiceSettingsFileName)
			err = ram.ReadValidate(serviceName, "ServiceSettings", serviceConfigFilePath, &instanceTriggerDeployment.Settings.Service)
			if err != nil {
				log.Fatal(err)
			}

			deployment = instanceTriggerDeployment
			err = deployment.Deploy()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	log.Println("done")
	return nil
}
