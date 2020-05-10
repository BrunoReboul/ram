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

package gcf

import (
	"fmt"
	"strings"

	"github.com/BrunoReboul/ram/utilities/ram"
	"github.com/google/uuid"
)

func (functionDeployment *FunctionDeployment) situate() (err error) {
	functionDeployment.Artifacts.CloudFunctionZipFullPath = fmt.Sprintf("./%s.zip", uuid.New())
	runTime, err := getRunTime(functionDeployment.Core.GoVersion)
	if err != nil {
		return err
	}

	functionDeployment.Artifacts.CloudFunction.AvailableMemoryMb = functionDeployment.Settings.Service.GCF.AvailableMemoryMb
	functionDeployment.Artifacts.CloudFunction.Description = functionDeployment.Settings.Service.GCF.Description
	functionDeployment.Artifacts.CloudFunction.EntryPoint = "EntryPoint"
	functionDeployment.Artifacts.CloudFunction.EventTrigger, err = functionDeployment.getEventTrigger()
	if err != nil {
		return err
	}
	functionDeployment.Artifacts.CloudFunction.Labels = map[string]string{"name": strings.ToLower(functionDeployment.Core.InstanceName)}
	functionDeployment.Artifacts.CloudFunction.Name = fmt.Sprintf("projects/%s/locations/%s/functions/%s",
		functionDeployment.Core.SolutionSettings.Hosting.ProjectID,
		functionDeployment.Core.SolutionSettings.Hosting.GCF.Region,
		functionDeployment.Core.InstanceName)
	functionDeployment.Artifacts.CloudFunction.Runtime = runTime
	functionDeployment.Artifacts.CloudFunction.ServiceAccountEmail = fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		functionDeployment.Core.ServiceName,
		functionDeployment.Core.SolutionSettings.Hosting.ProjectID)
	functionDeployment.Artifacts.CloudFunction.Timeout = functionDeployment.Settings.Service.GCF.Timeout
	functionDeployment.Artifacts.CloudFunction.IngressSettings = "ALLOW_ALL"

	functionDeployment.Artifacts.ZipFiles = make(map[string]string)
	functionGoContent, err := functionDeployment.makeFunctionGoContent()
	if err != nil {
		return err
	}
	functionDeployment.Artifacts.ZipFiles["function.go"] = functionGoContent
	functionDeployment.Artifacts.ZipFiles["go.mod"] = functionDeployment.makeGoModContent()
	functionDeployment.Artifacts.ZipFiles[ram.SettingsFileName] = functionDeployment.Artifacts.InstanceDeploymentYAMLContent

	if functionDeployment.Core.Commands.Dumpsettings {
		err := ram.MarshalYAMLWrite(fmt.Sprintf("%s/%s", functionDeployment.Core.RepositoryPath, "function_deployment.yaml"), functionDeployment)
		if err != nil {
			return err
		}
	}
	return nil
}
