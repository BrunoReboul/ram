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

	"github.com/BrunoReboul/ram/services/convertlog2feed"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureConvertlog2feedOrganizations
func (deployment *Deployment) configureConvertlog2feedOrganizations() (err error) {
	serviceName := "convertlog2feed"
	// Case activity group
	sinkNameSuffix := "activity-group"

	log.Printf("configure %s %s", serviceName, sinkNameSuffix)
	var convertlog2feedInstanceDeployment convertlog2feed.InstanceDeployment
	convertlog2feedInstance := convertlog2feedInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		convertlog2feedInstance.GCF.TriggerTopic = fmt.Sprintf("log-org%s-%s", organizationID, sinkNameSuffix)

		var directoryCustomerID string
		organization, err := deployment.Core.Services.CloudresourcemanagerService.Organizations.Get(fmt.Sprintf("organizations/%s", organizationID)).Context(deployment.Core.Ctx).Do()
		if err != nil {
			log.Printf("WARNING - cloudresourcemanagerService.Organizations.Get %v", err)
		} else {
			directoryCustomerID = organization.Owner.DirectoryCustomerId
		}
		convertlog2feedInstance.GCI.SuperAdminEmail = deployment.Core.SolutionSettings.Monitoring.DirectoryCustomerIDs[directoryCustomerID].SuperAdminEmail

		instanceFolderPath := strings.Replace(
			fmt.Sprintf("%s/%s_org%s_%s",
				instancesFolderPath,
				serviceName,
				organizationID,
				sinkNameSuffix), "-", "_", -1)
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), convertlog2feedInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
