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

	"github.com/BrunoReboul/ram/services/setlogsinks"
	"github.com/BrunoReboul/ram/utilities/ram"
)

// configureLogSinksOrganizations
func (deployment *Deployment) configureLogSinksOrganizations() (err error) {
	serviceName := "setlogsinks"
	// Case activity group
	sinkNameSuffix := "activity-group"
	filter := `resource.type="audited_resource" AND logName:"logs/cloudaudit.googleapis.com%2Factivity" AND protoPayload.serviceName="admin.googleapis.com" AND protoPayload.methodName:"group"`

	log.Printf("configure %s %s", serviceName, sinkNameSuffix)
	var setlogsinksInstanceDeployment setlogsinks.InstanceDeployment
	setlogsinksInstance := setlogsinksInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, ram.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, ram.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	for _, organizationID := range deployment.Core.SolutionSettings.Monitoring.OrganizationIDs {
		setlogsinksInstance.LSK.Parent = fmt.Sprintf("organizations/%s", organizationID)
		setlogsinksInstance.LSK.SinkNameSuffix = sinkNameSuffix
		setlogsinksInstance.LSK.Filter = filter
		setlogsinksInstance.LSK.TopicName = fmt.Sprintf("log-org%s-%s", organizationID, sinkNameSuffix)

		instanceFolderPath := strings.Replace(
			fmt.Sprintf("%s/%s_org%s_%s",
				instancesFolderPath,
				serviceName,
				organizationID,
				sinkNameSuffix), "-", "_", -1)
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ram.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, ram.InstanceSettingsFileName), setlogsinksInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
