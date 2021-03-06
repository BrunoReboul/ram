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
	"fmt"
	"log"
	"os"

	"github.com/BrunoReboul/ram/services/getgroupsettings"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureGetGroupSettingsDirectories
func (deployment *Deployment) configureGetGroupSettingsDirectories() (err error) {
	serviceName := "getgroupsettings"
	log.Printf("configure %s directories", serviceName)
	var getgroupsettingsInstanceDeployment getgroupsettings.InstanceDeployment
	getgroupsettingsInstance := getgroupsettingsInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	for directoryCustomerID, directorySettings := range deployment.Core.SolutionSettings.Monitoring.DirectoryCustomerIDs {
		getgroupsettingsInstance.GCI.SuperAdminEmail = directorySettings.SuperAdminEmail
		getgroupsettingsInstance.GCF.TriggerTopic = fmt.Sprintf("gci-groups-%s", directoryCustomerID)

		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_directory_%s",
			serviceName,
			directoryCustomerID))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), getgroupsettingsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
