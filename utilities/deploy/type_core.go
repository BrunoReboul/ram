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

package deploy

import (
	"context"

	"google.golang.org/api/appengine/v1"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/sourcerepo/v1"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	scheduler "cloud.google.com/go/scheduler/apiv1"
	"cloud.google.com/go/storage"
	"github.com/BrunoReboul/ram/utilities/solution"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/monitoring/v1"
	"google.golang.org/api/serviceusage/v1"
)

// Core structure common to all deployments
type Core struct {
	SolutionSettings            solution.Settings
	Ctx                         context.Context `yaml:"-"`
	EnvironmentName             string
	InstanceName                string
	ServiceName                 string
	AssetType                   string
	ProjectNumber               int64
	RepositoryPath              string
	RAMVersion                  string
	GoVersion                   string
	RamcliServiceAccount        string
	Dump                        bool
	InstanceFolderRelativePaths []string `yaml:"-"`
	Services                    struct {
		AppengineAPIService           *appengine.APIService           `yaml:"-"`
		AssetClient                   *asset.Client                   `yaml:"-"`
		BigqueryClient                *bigquery.Client                `yaml:"-"`
		CloudSchedulerClient          *scheduler.CloudSchedulerClient `yaml:"-"`
		Cloudbillingservice           *cloudbilling.APIService        `yaml:"-"`
		CloudbuildService             *cloudbuild.Service             `yaml:"-"`
		CloudfunctionsService         *cloudfunctions.Service         `yaml:"-"`
		CloudresourcemanagerService   *cloudresourcemanager.Service   `yaml:"-"`
		CloudresourcemanagerServicev2 *cloudresourcemanagerv2.Service `yaml:"-"`
		FirestoreClient               *firestore.Client               `yaml:"-"`
		IAMService                    *iam.Service                    `yaml:"-"`
		LoggingService                *logging.Service                `yaml:"-"`
		MonitoringService             *monitoring.Service             `yaml:"-"`
		PubsubPublisherClient         *pubsub.PublisherClient         `yaml:"-"`
		ServiceusageService           *serviceusage.Service           `yaml:"-"`
		SourcerepoService             *sourcerepo.Service             `yaml:"-"`
		StorageClient                 *storage.Client                 `yaml:"-"`
	} `yaml:"-"`
	Commands struct {
		// Makeyaml     bool
		Initialize          bool
		ConfigureAssetTypes bool
		MakeReleasePipeline bool
		Deploy              bool
		Check               bool
		Dumpsettings        bool
	} `yaml:"-"`
}
