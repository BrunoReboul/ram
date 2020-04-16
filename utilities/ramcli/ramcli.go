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
	"log"

	"github.com/BrunoReboul/ram/utilities/ram"

	"github.com/BrunoReboul/ram/services/publish2fs"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/option"
)

// Global structure for global variables
type Global struct {
	ctx                               context.Context
	projectsTriggersService           *cloudbuild.ProjectsTriggersService
	projectsLocationsFunctionsService *cloudfunctions.ProjectsLocationsFunctionsService
	settings                          settings
}

// Settings is the full set of parameters
type settings struct {
	Commands struct {
		Makeyaml     bool
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
	cloudfunctionsProjectsService := cloudfunctionsService.Projects
	global.projectsLocationsFunctionsService = cloudfunctionsProjectsService.Locations.Functions

	cloudbuildService, err = cloudbuild.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	cloudbuildProjectsService := cloudbuildService.Projects
	global.projectsTriggersService = cloudbuildProjectsService.Triggers
}

// RAMCli Real-time Asset Monitor cli
func RAMCli(global *Global) (err error) {
	global.settings.CheckArguments()
	log.Printf("Found %d instances", len(global.settings.InstanceFolderRelativePaths))

	if global.settings.Commands.Deploy {
		var deployment ram.MicroServiceInstanceDeployment
		for _, instanceFolderRelativePath := range global.settings.InstanceFolderRelativePaths {
			serviceName, instanceName := GetServiceAndInstanceNames(instanceFolderRelativePath)
			switch serviceName {
			case "publish2fs":
				deployment = publish2fs.NewDeployment()
				err := deployment.Deploy(global.settings.Versions.Go,
					global.settings.Versions.RAM,
					global.settings.RepositoryPath,
					global.settings.EnvironmentName,
					instanceName,
					global.settings.Commands.Dumpsettings)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	return nil
}
