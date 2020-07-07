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
	"flag"
	"fmt"
	"log"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// CheckArguments check cli arguments and build the list of microservices instances
func (deployment *Deployment) CheckArguments() {
	deployment.Core.GoVersion, deployment.Core.RAMVersion = getVersions()
	// flag.BoolVar(&settings.Commands.Makeyaml, "migrate-to-yaml", false, "make yaml settings files for setting.sh file")
	flag.BoolVar(&deployment.Core.Commands.Initialize, "init", false, "initial setup to be launched first, before manual, aka not automatable setup tasks")
	flag.BoolVar(&deployment.Core.Commands.ConfigureAssetTypes, "config", false, "For assets types defined in solution.yaml writes setfeeds, dumpinventory, stream2bq instance.yaml files and subfolders")
	flag.BoolVar(&deployment.Core.Commands.MakeReleasePipeline, "pipe", false, "make release pipeline using cloud build to deploy one instance, one microservice, or all")
	flag.BoolVar(&deployment.Core.Commands.Deploy, "deploy", false, "deploy one microservice instance")
	flag.BoolVar(&deployment.Core.Commands.Dumpsettings, "dump", false, fmt.Sprintf("dump all settings in %s", solution.SettingsFileName))
	flag.StringVar(&deployment.Core.RepositoryPath, "repo", ".", "Path to the root of the code repository")
	flag.StringVar(&deployment.Core.RamcliServiceAccount, "ramclisa", "", "Email of Service Account used when running ramcli")
	var assetType = flag.String("asset", "", "asset type e.g. k8s.io/Pod")
	var microserviceFolderName = flag.String("service", "", "Microservice folder name")
	var instanceFolderName = flag.String("instance", "", "Instance folder name")
	flag.StringVar(&deployment.Core.EnvironmentName, "environment", solution.DevelopmentEnvironmentName, "Environment name")
	flag.Parse()

	// case one instance
	if *instanceFolderName != "" {
		if *microserviceFolderName == "" {
			log.Fatalln("Missing service argument")
		}
		instanceRelativePath := fmt.Sprintf("%s/%s/%s/%s", solution.MicroserviceParentFolderName, *microserviceFolderName, solution.InstancesFolderName, *instanceFolderName)
		deployment.Core.InstanceFolderRelativePaths = []string{instanceRelativePath}
		instancePath := fmt.Sprintf("%s/%s", deployment.Core.RepositoryPath, instanceRelativePath)
		ffo.CheckPath(instancePath)
		return
	}

	if *microserviceFolderName != "" {
		// case one microservice
		deployment.Core.InstanceFolderRelativePaths = ffo.GetChild(deployment.Core.RepositoryPath, fmt.Sprintf("%s/%s/%s", solution.MicroserviceParentFolderName, *microserviceFolderName, solution.InstancesFolderName))
	} else {
		// case one assetType
		if *assetType != "" {
			deployment.Core.AssetType = *assetType
			return
		}
		// case all
		for _, microserviceRelativeFolderPath := range ffo.GetChild(deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName) {
			instanceFolderRelativePaths := ffo.GetChild(deployment.Core.RepositoryPath, fmt.Sprintf("%s/%s", microserviceRelativeFolderPath, solution.InstancesFolderName))
			for _, instanceFolderRelativePath := range instanceFolderRelativePaths {
				deployment.Core.InstanceFolderRelativePaths = append(deployment.Core.InstanceFolderRelativePaths, instanceFolderRelativePath)
			}
		}
	}
	if len(deployment.Core.InstanceFolderRelativePaths) == 0 {
		log.Fatalln("No instance found")
	}
	return
}
