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
	"strings"

	"github.com/BrunoReboul/ram/services/setfeeds"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureSetFeedsAssetTypes for assets types defined in solution.yaml writes setfeeds instance.yaml files and subfolders
func (deployment *Deployment) configureSetFeedsAssetTypes() (err error) {
	serviceName := "setfeeds"
	log.Printf("configure %s asset types", serviceName)
	var setfeedsInstanceDeployment setfeeds.InstanceDeployment
	setfeedsInstance := setfeedsInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		setfeedsInstance.CAI.Parent = fmt.Sprintf("organizations/%s", organizationID)

		// one and only one iam policy feed for all asset types
		setfeedsInstance.CAI.ContentType = "IAM_POLICY"
		setfeedsInstance.CAI.AssetTypes = deployment.Core.SolutionSettings.Monitoring.AssetTypes.IAMPolicies
		instanceFolderPath := fmt.Sprintf("%s/%s_org%s_iam_policies", instancesFolderPath, serviceName, organizationID)
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), setfeedsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)

		// one resource feed per asset type
		for _, assetType := range deployment.Core.SolutionSettings.Monitoring.AssetTypes.Resources {
			setfeedsInstance.CAI.ContentType = "RESOURCE"
			setfeedsInstance.CAI.AssetTypes = []string{assetType}
			instanceFolderPath := strings.Replace(
				fmt.Sprintf("%s/%s_org%s_%s",
					instancesFolderPath,
					serviceName,
					organizationID,
					cai.GetAssetShortTypeName(assetType)), "-", "_", -1)
			if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
				os.Mkdir(instanceFolderPath, 0755)
			}
			if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), setfeedsInstance); err != nil {
				return err
			}
			log.Printf("done %s", instanceFolderPath)
		}
	}
	return nil
}
