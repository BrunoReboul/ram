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

	"github.com/BrunoReboul/ram/services/publish2fs"
	"github.com/BrunoReboul/ram/utilities/deploy"

	"github.com/BrunoReboul/ram/utilities/ram"

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
	core deploy.Core
}

// Initialize is to be executed in the init()
func Initialize(ctx context.Context, global *Global) {
	global.core.Ctx = ctx
	var err error
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("ERROR - google.FindDefaultCredentials %v", err)
	}
	global.core.Services.CloudbuildService, err = cloudbuild.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.core.Services.CloudfunctionsService, err = cloudfunctions.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.core.Services.CloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.core.Services.IAMService, err = iam.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	global.core.Services.ServiceusageService, err = serviceusage.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
}

// RAMCli Real-time Asset Monitor cli
func RAMCli(global *Global) (err error) {
	var deployment ram.Deployment
	global.CheckArguments()

	solutionConfigFilePath := fmt.Sprintf("%s/%s", global.core.RepositoryPath, ram.SolutionSettingsFileName)
	err = ram.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &global.core.SolutionSettings)
	if err != nil {
		log.Fatal(err)
	}
	global.core.SolutionSettings.Situate(global.core.EnvironmentName)
	global.core.ProjectNumber, err = getProjectNumber(global.core.Ctx, global.core.Services.CloudresourcemanagerService, global.core.SolutionSettings.Hosting.ProjectID)
	log.Printf("found %d instance(s)", len(global.core.InstanceFolderRelativePaths))

	if global.core.Commands.Deploy {
		for _, instanceFolderRelativePath := range global.core.InstanceFolderRelativePaths {
			global.core.ServiceName, global.core.InstanceName = GetServiceAndInstanceNames(instanceFolderRelativePath)

			switch global.core.ServiceName {
			case "publish2fs":
				instanceDeployment := publish2fs.NewInstanceDeployment()
				instanceDeployment.Core = &global.core
				deployment = instanceDeployment
				if err := deployment.Deploy(); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	if global.core.Commands.Maketrigger {
		instanceTriggerDeployment := NewInstanceTrigger()
		instanceTriggerDeployment.Core = &global.core

		for _, instanceFolderRelativePath := range global.core.InstanceFolderRelativePaths {
			global.core.ServiceName, global.core.InstanceName = GetServiceAndInstanceNames(instanceFolderRelativePath)
			serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", global.core.RepositoryPath, ram.MicroserviceParentFolderName, global.core.ServiceName, ram.ServiceSettingsFileName)
			if err = ram.ReadValidate(global.core.ServiceName, "ServiceSettings", serviceConfigFilePath,
				&instanceTriggerDeployment.Settings.Service); err != nil {
				log.Fatal(err)
			}
			deployment = instanceTriggerDeployment
			if err = deployment.Deploy(); err != nil {
				log.Fatal(err)
			}
		}
	}
	log.Println("ramcli done")
	return nil
}
