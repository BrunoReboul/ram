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

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"github.com/BrunoReboul/ram/services/publish2fs"

	"github.com/BrunoReboul/ram/utilities/ram"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
)

// Initialize is to be executed in the init()
func Initialize(ctx context.Context, deployment *Deployment) {
	deployment.Core.Ctx = ctx
	var err error
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("ERROR - google.FindDefaultCredentials %v", err)
	}
	deployment.Core.Services.Cloudbillingservice, err = cloudbilling.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.CloudbuildService, err = cloudbuild.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.CloudfunctionsService, err = cloudfunctions.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.CloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.CloudresourcemanagerServicev2, err = cloudresourcemanagerv2.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.IAMService, err = iam.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.ServiceusageService, err = serviceusage.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.PubsubPublisherClient, err = pubsub.NewPublisherClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}

// RAMCli Real-time Asset Monitor cli
func RAMCli(deployment *Deployment) (err error) {
	deployment.CheckArguments()
	solutionConfigFilePath := fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, ram.SolutionSettingsFileName)
	err = ram.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &deployment.Core.SolutionSettings)
	if err != nil {
		log.Fatal(err)
	}
	deployment.Core.SolutionSettings.Situate(deployment.Core.EnvironmentName)
	deployment.Core.ProjectNumber, err = getProjectNumber(deployment.Core.Ctx, deployment.Core.Services.CloudresourcemanagerService, deployment.Core.SolutionSettings.Hosting.ProjectID)
	log.Printf("found %d instance(s)", len(deployment.Core.InstanceFolderRelativePaths))

	for _, instanceFolderRelativePath := range deployment.Core.InstanceFolderRelativePaths {
		deployment.Core.ServiceName, deployment.Core.InstanceName = GetServiceAndInstanceNames(instanceFolderRelativePath)
		switch deployment.Core.ServiceName {
		case "publish2fs":
			deployment.deployPublish2fs()
		}
	}
	log.Println("ramcli done")
	return nil
}

func (deployment *Deployment) deployPublish2fs() {
	instanceDeployment := publish2fs.NewInstanceDeployment()
	instanceDeployment.Core = &deployment.Core
	err := instanceDeployment.ReadValidate()
	if err != nil {
		log.Fatal(err)
	}
	instanceDeployment.Situate()
	if deployment.Core.Commands.Maketrigger {
		deployment.Settings.Service.GCB = instanceDeployment.Settings.Service.GCB
		deployment.Settings.Service.GSU = instanceDeployment.Settings.Service.GSU
		err = deployment.deployInstanceTrigger()
	} else {
		if deployment.Core.Commands.Deploy {
			err = instanceDeployment.Deploy()
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
