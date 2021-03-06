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

package stream2bq

import (
	"fmt"
	"os"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// ReadValidate reads and validates service and instance settings
func (instanceDeployment *InstanceDeployment) ReadValidate() (err error) {
	serviceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s", instanceDeployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, instanceDeployment.Core.ServiceName, solution.ServiceSettingsFileName)
	if _, err := os.Stat(serviceConfigFilePath); !os.IsNotExist(err) {
		err = ffo.ReadValidate(instanceDeployment.Core.ServiceName, "ServiceSettings", serviceConfigFilePath, &instanceDeployment.Settings.Service)
		if err != nil {
			return err
		}
	}
	instanceConfigFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", instanceDeployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, instanceDeployment.Core.ServiceName, solution.InstancesFolderName, instanceDeployment.Core.InstanceName, solution.InstanceSettingsFileName)
	if _, err := os.Stat(instanceConfigFilePath); !os.IsNotExist(err) {
		err = ffo.ReadValidate(instanceDeployment.Core.InstanceName, "InstanceSettings", instanceConfigFilePath, &instanceDeployment.Settings.Instance)
		if err != nil {
			return err
		}
	}
	return nil
}
