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

	"github.com/BrunoReboul/ram/utilities/ram"
)

// CheckArguments() check cli arguments and build the list of microservices instances
func (settings *settings) CheckArguments() {
	settings.Versions.Go, settings.Versions.RAM = GetVersions()
	flag.BoolVar(&settings.Commands.Makeyaml, "migrate-to-yaml", false, "make yaml settings files for setting.sh file")
	flag.BoolVar(&settings.Commands.Maketrigger, "make-trigger", false, "make cloud build triggers to deploy one instance, one microservice, or all")
	flag.BoolVar(&settings.Commands.Deploy, "deploy", false, "deploy one microservice instance")
	flag.BoolVar(&settings.Commands.Dumpsettings, "dump", false, fmt.Sprintf("dump all settings in %s", ram.SettingsFileName))
	flag.StringVar(&settings.RepositoryPath, "repo", ".", "Path to the root of the code repository")
	var microserviceFolderName = flag.String("service", "", "Microservice folder name")
	var instanceFolderName = flag.String("instance", "", "Instance folder name")
	flag.StringVar(&settings.EnvironmentName, "environment", ram.DevelopmentEnvironmentName, "Environment name")
	flag.Parse()

	// case the one instance
	if *instanceFolderName != "" {
		if *microserviceFolderName == "" {
			log.Fatalln("Missing service argument")
		}
		instanceRelativePath := fmt.Sprintf("%s/%s/%s/%s", ram.MicroserviceParentFolderName, *microserviceFolderName, ram.InstancesFolderName, *instanceFolderName)
		instancePath := fmt.Sprintf("%s/%s", settings.RepositoryPath, instanceRelativePath)
		ram.CheckPath(instancePath)
		settings.InstanceFolderRelativePaths = []string{instanceRelativePath}
		return
	}

	if *microserviceFolderName != "" {
		// case the one microservice
		settings.InstanceFolderRelativePaths = ram.GetChild(settings.RepositoryPath, fmt.Sprintf("%s/%s/%s", ram.MicroserviceParentFolderName, *microserviceFolderName, ram.InstancesFolderName))
	} else {
		// case all
		for _, microserviceRelativeFolderPath := range ram.GetChild(settings.RepositoryPath, ram.MicroserviceParentFolderName) {
			instanceFolderRelativePaths := ram.GetChild(settings.RepositoryPath, fmt.Sprintf("%s/%s", microserviceRelativeFolderPath, ram.InstancesFolderName))
			for _, instanceFolderRelativePath := range instanceFolderRelativePaths {
				settings.InstanceFolderRelativePaths = append(settings.InstanceFolderRelativePaths, instanceFolderRelativePath)
			}
		}
	}
	if len(settings.InstanceFolderRelativePaths) == 0 {
		log.Fatalln("No instance found")
	}
	return
}
