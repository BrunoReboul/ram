// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE_2.0
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

	"github.com/BrunoReboul/ram/services/stream2bq"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureStream2bqAssetTypes for assets types defined in solution.yaml writes stream2bq instance.yaml files and subfolders
func (deployment *Deployment) configureStream2bqAssetTypes() (err error) {
	serviceName := "stream2bq"
	log.Printf("configure %s asset types", serviceName)
	var stream2bqInstanceDeployment stream2bq.InstanceDeployment
	stream2bqInstance := stream2bqInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	// violations, complianceStatus
	for _, tableName := range []string{"violations", "complianceStatus"} {
		stream2bqInstance.Bigquery.TableName = tableName
		stream2bqInstance.GCF.TriggerTopic = fmt.Sprintf("ram-%s", tableName)
		instanceFolderPath := fmt.Sprintf("%s/%s_%s",
			instancesFolderPath,
			serviceName,
			tableName)
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), stream2bqInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}

	// assets
	for _, assetType := range deployment.Core.SolutionSettings.Monitoring.AssetTypes.Resources {
		stream2bqInstance.Bigquery.TableName = "assets"
		assetShortName := cai.GetAssetShortTypeName(assetType)
		stream2bqInstance.GCF.TriggerTopic = fmt.Sprintf("cai-rces-%s", assetShortName)
		instanceFolderPath := strings.Replace(
			fmt.Sprintf("%s/%s_rces_%s",
				instancesFolderPath,
				serviceName,
				cai.GetAssetShortTypeName(assetType)), "-", "_", -1)
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), stream2bqInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
