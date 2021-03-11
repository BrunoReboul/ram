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

	"github.com/BrunoReboul/ram/services/upload2gcs"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureUpload2gcsMetadataTypes for assets types defined in solution.yaml writes upload2gcs instance.yaml files and subfolders
func (deployment *Deployment) configureUpload2gcsMetadataTypes() (err error) {
	serviceName := "upload2gcs"
	log.Printf("configure %s asset types", serviceName)
	var upload2gcsInstanceDeployment upload2gcs.InstanceDeployment
	upload2gcsInstance := upload2gcsInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	// assets
	for _, assetType := range deployment.Core.SolutionSettings.Monitoring.AssetTypes.Resources {
		assetShortName := cai.GetAssetShortTypeName(assetType)
		upload2gcsInstance.GCF.TriggerTopic = fmt.Sprintf("cai-rces-%s", assetShortName)
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_rces_%s",
			serviceName,
			cai.GetAssetShortTypeName(assetType)))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), upload2gcsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}

	// iam policy
	upload2gcsInstance.GCF.TriggerTopic = deployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.IAMPolicies
	instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_iam_policies",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), upload2gcsInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)

	// groups by directory
	for directoryCustomerID := range deployment.Core.SolutionSettings.Monitoring.DirectoryCustomerIDs {
		upload2gcsInstance.GCF.TriggerTopic = fmt.Sprintf("gci-groups-%s", directoryCustomerID)
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_%s",
			serviceName,
			upload2gcsInstance.GCF.TriggerTopic))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), upload2gcsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}

	// group membership

	for _, topicName := range []string{deployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupMembers,
		deployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.GCIGroupSettings} {
		upload2gcsInstance.GCF.TriggerTopic = topicName
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_%s",
			serviceName,
			topicName))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), upload2gcsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}

	return nil
}
