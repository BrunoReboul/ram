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

	"github.com/BrunoReboul/ram/services/splitdump"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureSplitDumpSingleInstance
func (deployment *Deployment) configureSplitDumpSingleInstance() (err error) {
	serviceName := "splitdump"
	log.Printf("configure %s single instance", serviceName)
	var splitdumpInstanceDeployment splitdump.InstanceDeployment
	splitdumpInstance := splitdumpInstanceDeployment.Settings.Instance
	serviceFolderPath := fmt.Sprintf("%s/%s/%s", deployment.Core.RepositoryPath, solution.MicroserviceParentFolderName, serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	// Default value
	splitdumpInstance.SplitThresholdLineNumber = 1000
	splitdumpInstance.ScannerBufferSizeKiloBytes = 128

	instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_single_instance",
		serviceName))
	if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
		os.Mkdir(instanceFolderPath, 0755)
	}
	if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s", instanceFolderPath, solution.InstanceSettingsFileName), splitdumpInstance); err != nil {
		return err
	}
	log.Printf("done %s", instanceFolderPath)
	return nil
}
