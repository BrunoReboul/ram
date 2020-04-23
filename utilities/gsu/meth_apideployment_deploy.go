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

package gsu

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/ram"

	"google.golang.org/api/serviceusage/v1"
)

var activeAPIs []string

// Deploy activates APIs
func (apiDeployment *APIDeployment) Deploy() (err error) {
	apiDeployment.Artifacts.ServicesService = apiDeployment.Artifacts.ServiceusageService.Services
	apiDeployment.Artifacts.OperationsService = apiDeployment.Artifacts.ServiceusageService.Operations

	activeAPIs = make([]string, 0)
	parent := fmt.Sprintf("projects/%s", apiDeployment.Core.SolutionSettings.Hosting.ProjectID)
	err = apiDeployment.Artifacts.ServicesService.List(parent).Filter("state:ENABLED").PageSize(200).Pages(apiDeployment.Core.Ctx, browseActiveAPIs)
	if err != nil {
		return err
	}

	for _, apiName := range apiDeployment.Settings.Service.GSU.APIList {
		if ram.Find(activeAPIs, apiName) {
			log.Printf("%s gsu API already active %s", apiDeployment.Core.InstanceName, apiName)
		} else {
			err = apiDeployment.activateAPI(apiName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func browseActiveAPIs(listServicesResponse *serviceusage.ListServicesResponse) (err error) {
	for _, googleAPIServiceusageV1Service := range listServicesResponse.Services {
		parts := strings.Split(googleAPIServiceusageV1Service.Name, "/")
		activeAPIs = append(activeAPIs, parts[len(parts)-1])
	}
	return nil
}

func (apiDeployment *APIDeployment) activateAPI(apiName string) (err error) {
	name := fmt.Sprintf("projects/%s/services/%s", apiDeployment.Core.SolutionSettings.Hosting.ProjectID, apiName)
	var request serviceusage.EnableServiceRequest
	operation, err := apiDeployment.Artifacts.ServicesService.Enable(name, &request).Context(apiDeployment.Core.Ctx).Do()
	if err != nil {
		return err
	}
	log.Printf("%s gsu API %s activation started", apiDeployment.Core.InstanceName, apiName)
	operationName := operation.Name
	log.Println(operationName)

	// Operation GET not working as expecter: returns 404 notFound

	// for {
	// 	time.Sleep(5 * time.Second)
	// 	operation, err = apiDeployment.Artifacts.OperationsService.Get(operationName).Context(apiDeployment.Core.Ctx).Do()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if operation.Done {
	// 		break
	// 	}
	// }
	// ram.JSONMarshalIndentPrint(operation)

	// Work arround: check result, API activated
	var googleAPIServiceusageV1Service *serviceusage.GoogleApiServiceusageV1Service
	for {
		time.Sleep(5 * time.Second)
		googleAPIServiceusageV1Service, err = apiDeployment.Artifacts.ServicesService.Get(name).Context(apiDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
		if googleAPIServiceusageV1Service.State == "ENABLED" {
			break
		}
	}
	// ram.JSONMarshalIndentPrint(googleAPIServiceusageV1Service)
	log.Printf("%s gsu API %s is active", apiDeployment.Core.InstanceName, apiName)

	return nil
}
