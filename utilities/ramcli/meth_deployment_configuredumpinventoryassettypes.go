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

	"github.com/BrunoReboul/ram/services/dumpinventory"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureDumpInventoryAssetTypes for assets types defined in solution.yaml writes dumpinventory instance.yaml files and subfolders
func (deployment *Deployment) configureDumpInventoryAssetTypes() (err error) {
	serviceName := "dumpinventory"
	log.Printf("configure %s asset types", serviceName)
	var dumpinventoryInstanceDeployment dumpinventory.InstanceDeployment
	dumpinventoryInstance := dumpinventoryInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		dumpinventoryInstance.CAI.Parent = fmt.Sprintf("organizations/%s", organizationID)
		dumpinventoryInstance.SCH.Schedulers = deployment.Core.SolutionSettings.Monitoring.DefaultSchedulers

		// one and only one iam policy feed for all asset types
		dumpinventoryInstance.CAI.ContentType = "IAM_POLICY"
		dumpinventoryInstance.CAI.AssetTypes = deployment.Core.SolutionSettings.Monitoring.AssetTypes.IAMPolicies
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_org%s_iam_policies",
			serviceName,
			organizationID))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), dumpinventoryInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)

		// one resource feed per asset type
		for _, assetType := range deployment.Core.SolutionSettings.Monitoring.AssetTypes.Resources {
			dumpinventoryInstance.CAI.ContentType = "RESOURCE"
			dumpinventoryInstance.CAI.AssetTypes = []string{assetType}
			instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_org%s_%s",
				serviceName,
				organizationID,
				cai.GetAssetShortTypeName(assetType)))
			if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
				os.Mkdir(instanceFolderPath, 0755)
			}
			if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), dumpinventoryInstance); err != nil {
				return err
			}
			log.Printf("done %s", instanceFolderPath)
		}
	}
	return nil
}
