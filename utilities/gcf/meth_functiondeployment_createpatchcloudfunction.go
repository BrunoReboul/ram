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
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"google.golang.org/api/cloudfunctions/v1"
)

// createPatchCloudFunction looks for an existing cloud function patch it if found else create it
func (functionDeployment *FunctionDeployment) createPatchCloudFunction() (err error) {
	var operation *cloudfunctions.Operation
	location := fmt.Sprintf("projects/%s/locations/%s", functionDeployment.Core.SolutionSettings.Hosting.ProjectID, functionDeployment.Core.SolutionSettings.Hosting.GCF.Region)
	retreivedCloudFunction, err := functionDeployment.Artifacts.ProjectsLocationsFunctionsService.Get(functionDeployment.Artifacts.CloudFunction.Name).Context(functionDeployment.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			operation, err = functionDeployment.Artifacts.ProjectsLocationsFunctionsService.Create(location,
				&functionDeployment.Artifacts.CloudFunction).Context(functionDeployment.Core.Ctx).Do()
			if err != nil {
				return fmt.Errorf("ProjectsLocationsFunctionsService.Create %v", err)
			}
		} else {
			return fmt.Errorf("ProjectsLocationsFunctionsService.Get %v", err)
		}
	} else {
		log.Printf("%s gcf patch existing cloud function %s", functionDeployment.Core.InstanceName, retreivedCloudFunction.Name)
		operation, err = functionDeployment.Artifacts.ProjectsLocationsFunctionsService.Patch(functionDeployment.Artifacts.CloudFunction.Name,
			&functionDeployment.Artifacts.CloudFunction).Context(functionDeployment.Core.Ctx).Do()
		if err != nil {
			return fmt.Errorf("ProjectsLocationsFunctionsService.Patch %v", err)
		}
	}

	name := operation.Name
	log.Printf("%s gcf cloud function deployment started", functionDeployment.Core.InstanceName)
	log.Println(name)
	for {
		time.Sleep(5 * time.Second)
		for i := 0; i < Retries; i++ {
			operation, err = functionDeployment.Artifacts.OperationsService.Get(name).Context(functionDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "500") && strings.Contains(err.Error(), "backendError") {
					log.Printf("%s ERROR getting operation status, iteration %d, wait 5 sec and retry %v", functionDeployment.Core.InstanceName, i, err)
					time.Sleep(5 * time.Second)
				} else {
					return err
				}
			}
		}
		if err != nil {
			return err
		}
		if operation.Done {
			break
		}
	}
	ffo.JSONMarshalIndentPrint(operation)
	if operation.Error != nil {
		return fmt.Errorf("Function deployment error %v", operation.Error)
	}
	return nil
}
