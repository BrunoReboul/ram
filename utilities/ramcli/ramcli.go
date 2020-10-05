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
	"os"
	"strings"

	asset "cloud.google.com/go/asset/apiv1"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	scheduler "cloud.google.com/go/scheduler/apiv1"

	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/appengine/v1"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"google.golang.org/api/sourcerepo/v1"
)

// Initialize is to be executed in the init()
func Initialize(ctx context.Context, deployment *Deployment) {
	deployment.Core.Ctx = ctx
	var err error
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("ERROR - google.FindDefaultCredentials %v", err)
	}
	deployment.Core.Services.AppengineAPIService, err = appengine.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.AssetClient, err = asset.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
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
	deployment.Core.Services.PubsubPublisherClient, err = pubsub.NewPublisherClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.SourcerepoService, err = sourcerepo.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.StorageClient, err = storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	deployment.Core.Services.CloudSchedulerClient, err = scheduler.NewCloudSchedulerClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
}

// RAMCli Real-time Asset Monitor cli
func RAMCli(deployment *Deployment) (err error) {
	err = deployment.CheckArguments()
	if err != nil {
		return err
	}
	log.Printf("goVersion %s, ramVersion %s", deployment.Core.GoVersion, deployment.Core.RAMVersion)

	solutionConfigFilePath := fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, solution.SolutionSettingsFileName)
	err = ffo.ReadValidate("", "SolutionSettings", solutionConfigFilePath, &deployment.Core.SolutionSettings)
	if err != nil {
		return err
	}
	deployment.Core.SolutionSettings.Situate(deployment.Core.EnvironmentName)
	deployment.Core.ProjectNumber, err = getProjectNumber(deployment.Core.Ctx, deployment.Core.Services.CloudresourcemanagerService, deployment.Core.SolutionSettings.Hosting.ProjectID)

	creds, err := google.FindDefaultCredentials(deployment.Core.Ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("ERROR - google.FindDefaultCredentials %v", err)
	}
	// BQ client cannot be initiated in the Intialize func as other clients as it requires the projdctID that is know only at this stage
	deployment.Core.Services.BigqueryClient, err = bigquery.NewClient(deployment.Core.Ctx, deployment.Core.SolutionSettings.Hosting.ProjectID, option.WithCredentials(creds))
	if err != nil {
		return err
	}

	if deployment.Core.AssetType != "" {
		// For one (new) assetType build the list of related instances to deploy accross services. aka transversal point of view
		// Cannot be done in checkarguments like for other deployments as requires orgIDs list that is available only after ReadValidate
		var instanceFolderRelativePaths []string
		for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
			serviceName := "setfeeds"
			instanceRelativePath := strings.Replace(
				fmt.Sprintf("%s/%s/%s/%s_org%s_%s",
					solution.MicroserviceParentFolderName,
					serviceName,
					solution.InstancesFolderName,
					serviceName,
					organizationID,
					cai.GetAssetShortTypeName(deployment.Core.AssetType)), "-", "_", -1)
			instancePath := fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, instanceRelativePath)
			if _, err := os.Stat(instancePath); err != nil {
				return err
			}
			instanceFolderRelativePaths = append(instanceFolderRelativePaths, instanceRelativePath)

			serviceName = "dumpinventory"
			instanceRelativePath = strings.Replace(
				fmt.Sprintf("%s/%s/%s/%s_org%s_%s",
					solution.MicroserviceParentFolderName,
					serviceName,
					solution.InstancesFolderName,
					serviceName,
					organizationID,
					cai.GetAssetShortTypeName(deployment.Core.AssetType)), "-", "_", -1)
			instancePath = fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, instanceRelativePath)
			if _, err := os.Stat(instancePath); err != nil {
				return err
			}
			instanceFolderRelativePaths = append(instanceFolderRelativePaths, instanceRelativePath)
		}
		serviceName := "stream2bq"
		instanceRelativePath := strings.Replace(
			fmt.Sprintf("%s/%s/%s/%s_rces_%s",
				solution.MicroserviceParentFolderName,
				serviceName,
				solution.InstancesFolderName,
				serviceName,
				cai.GetAssetShortTypeName(deployment.Core.AssetType)), "-", "_", -1)
		instancePath := fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, instanceRelativePath)
		if _, err := os.Stat(instancePath); err != nil {
			return err
		}
		instanceFolderRelativePaths = append(instanceFolderRelativePaths, instanceRelativePath)

		serviceName = "upload2gcs"
		instanceRelativePath = strings.Replace(
			fmt.Sprintf("%s/%s/%s/%s_rces_%s",
				solution.MicroserviceParentFolderName,
				serviceName,
				solution.InstancesFolderName,
				serviceName,
				cai.GetAssetShortTypeName(deployment.Core.AssetType)), "-", "_", -1)
		instancePath = fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, instanceRelativePath)
		if _, err := os.Stat(instancePath); err != nil {
			return err
		}
		instanceFolderRelativePaths = append(instanceFolderRelativePaths, instanceRelativePath)

		deployment.Core.InstanceFolderRelativePaths = instanceFolderRelativePaths
	}

	switch true {
	case deployment.Core.Commands.Initialize:
		if err = deployment.initialize(); err != nil {
			return err
		}
	case deployment.Core.Commands.ConfigureAssetTypes:
		if err = deployment.configureSetFeedsAssetTypes(); err != nil {
			return err
		}
		if err = deployment.configureDumpInventoryAssetTypes(); err != nil {
			return err
		}
		if err = deployment.configureSplitDumpSingleInstance(); err != nil {
			return err
		}
		if err = deployment.configurePublish2fsInstances(); err != nil {
			return err
		}
		if err = deployment.configureStream2bqAssetTypes(); err != nil {
			return err
		}
		if err = deployment.configureUpload2gcsMetadataTypes(); err != nil {
			return err
		}
		if err = deployment.configureListGroupsDirectories(); err != nil {
			return err
		}
		if err = deployment.configureListGroupMembersDirectories(); err != nil {
			return err
		}
		if err = deployment.configureGetGroupSettingsDirectories(); err != nil {
			return err
		}
		if err = deployment.configureLogSinksOrganizations(); err != nil {
			return err
		}
		if err = deployment.configureConvertlog2feedOrganizations(); err != nil {
			return err
		}
	default:
		log.Printf("found %d instance(s)", len(deployment.Core.InstanceFolderRelativePaths))
		for _, instanceFolderRelativePath := range deployment.Core.InstanceFolderRelativePaths {
			deployment.Core.ServiceName, deployment.Core.InstanceName = getServiceAndInstanceNames(instanceFolderRelativePath)
			switch deployment.Core.ServiceName {
			case "setfeeds":
				if err = deployment.deploySetFeeds(); err != nil {
					return err
				}
			case "dumpinventory":
				if err = deployment.deployDumpInventory(); err != nil {
					return err
				}
			case "splitdump":
				if err = deployment.deploySplitDump(); err != nil {
					return err
				}
			case "publish2fs":
				if err = deployment.deployPublish2fs(); err != nil {
					return err
				}
			case "monitor":
				if err = deployment.deployMonitor(); err != nil {
					return err
				}
			case "stream2bq":
				if err = deployment.deployStream2bq(); err != nil {
					return err
				}
			case "upload2gcs":
				if err = deployment.deployUpload2gcs(); err != nil {
					return err
				}
			case "listgroups":
				if err = deployment.deployListGroups(); err != nil {
					return err
				}
			case "listgroupmembers":
				if err = deployment.deployListGroupMembers(); err != nil {
					return err
				}
			case "getgroupsettings":
				if err = deployment.deployGetGroupSettings(); err != nil {
					return err
				}
			case "setlogsinks":
				if err = deployment.deploySetLogSinks(); err != nil {
					return err
				}
			case "convertlog2feed":
				if err = deployment.deployConvertLog2Feed(); err != nil {
					return err
				}
			}
		}
	}
	log.Println("ramcli done")
	return nil
}
