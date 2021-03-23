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

	"github.com/BrunoReboul/ram/services/publish2fs"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configurePublish2fsInstances
func (deployment *Deployment) configurePublish2fsInstances() (err error) {
	serviceName := "publish2fs"
	log.Printf("configure %s instances", serviceName)
	var publish2fsInstanceDeployment publish2fs.InstanceDeployment
	publish2fsInstance := publish2fsInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	publish2fsInstance.GCF.TriggerTopic = "cai-rces-cloudresourcemanager-Organization"
	instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_cloudresourcemanager_Organization",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), publish2fsInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)

	publish2fsInstance.GCF.TriggerTopic = "cai-rces-cloudresourcemanager-Folder"
	instanceFolderPath = makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_cloudresourcemanager_Folder",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), publish2fsInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)

	publish2fsInstance.GCF.TriggerTopic = "cai-rces-cloudresourcemanager-Project"
	instanceFolderPath = makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_cloudresourcemanager_Project",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), publish2fsInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)

	publish2fsInstance.GCF.TriggerTopic = "gci-groupMembers"
	instanceFolderPath = makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_gci_groupMembers",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), publish2fsInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)

	for directoryCustomerID := range deployment.Core.SolutionSettings.Monitoring.DirectoryCustomerIDs {
		publish2fsInstance.GCF.TriggerTopic = fmt.Sprintf("gci-groups-%s", directoryCustomerID)
		instanceFolderPath = makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_gci_groups_%s",
			serviceName,
			directoryCustomerID))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), publish2fsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
