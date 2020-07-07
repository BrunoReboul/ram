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
	"log"
	"os"

	"github.com/BrunoReboul/ram/utilities/ffo"
)

// Retries is the max number of tentative to get the operation status on the cloud function deployment, to deal with transient
const Retries = 5

// Deploy create or update a cloud function
func (functionDeployment *FunctionDeployment) Deploy() (err error) {
	functionDeployment.Artifacts.ProjectsLocationsFunctionsService = functionDeployment.Core.Services.CloudfunctionsService.Projects.Locations.Functions
	functionDeployment.Artifacts.OperationsService = functionDeployment.Core.Services.CloudfunctionsService.Operations
	err = functionDeployment.situate()
	if err != nil {
		return err
	}
	log.Printf("%s gcf situate settings done", functionDeployment.Core.InstanceName)
	err = ffo.ZipSource(functionDeployment.Artifacts.CloudFunctionZipFullPath, functionDeployment.Artifacts.ZipFiles)
	if err != nil {
		return err
	}
	log.Printf("%s gcf sources zipped", functionDeployment.Core.InstanceName)
	err = functionDeployment.getUploadURL()
	if err != nil {
		return err
	}
	log.Printf("%s gcf signed URL for upload retreived", functionDeployment.Core.InstanceName)
	response, err := functionDeployment.UploadZipUsingSignedURL()
	if err != nil {
		return err
	}
	log.Printf("%s gcf upload %s response status code: %v", functionDeployment.Core.InstanceName, functionDeployment.Artifacts.CloudFunctionZipFullPath, response.StatusCode)
	err = functionDeployment.createPatchCloudFunction()
	if err != nil {
		return err
	}
	log.Printf("%s gcf function created or patched", functionDeployment.Core.InstanceName)
	err = os.Remove(functionDeployment.Artifacts.CloudFunctionZipFullPath)
	if err != nil {
		return err
	}
	log.Printf("%s gcf file removed %s", functionDeployment.Core.InstanceName, functionDeployment.Artifacts.CloudFunctionZipFullPath)
	return nil
}
