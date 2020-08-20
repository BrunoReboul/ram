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

package gae

import (
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/api/appengine/v1"
)

// Deploy AppDeployment check if the repo exist, if not try to create it
func (appDeployment *AppDeployment) Deploy() (err error) {
	log.Printf("%s gae application engine", appDeployment.Core.InstanceName)
	appsService := appDeployment.Core.Services.AppengineAPIService.Apps
	appsOperationsService := appengine.NewAppsOperationsService(appDeployment.Core.Services.AppengineAPIService)
	app, err := appsService.Get(appDeployment.Core.SolutionSettings.Hosting.ProjectID).Context(appDeployment.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
			var appToCreate appengine.Application
			appToCreate.Id = appDeployment.Core.SolutionSettings.Hosting.ProjectID
			appToCreate.LocationId = appDeployment.Core.SolutionSettings.Hosting.GAE.Region
			operation, err := appsService.Create(&appToCreate).Context(appDeployment.Core.Ctx).Do()
			if err != nil {
				if strings.Contains(err.Error(), "403") {
					log.Printf("%s gae WARNING impossible to CREATE application %v", appDeployment.Core.InstanceName, err)
					return nil
				}
				return fmt.Errorf("gae appsService.Create %v", err)
			}
			log.Printf("%s gae application deployment started %s", appDeployment.Core.InstanceName, appToCreate.Id)
			log.Println(operation.Name)
			parts := strings.Split(operation.Name, "/")
			operationID := parts[len(parts)-1]
			for {
				time.Sleep(5 * time.Second)
				operation, err = appsOperationsService.Get(appDeployment.Core.SolutionSettings.Hosting.ProjectID, operationID).Context(appDeployment.Core.Ctx).Do()
				if err != nil {
					return fmt.Errorf("gae appsOperationsService.Get %v", err)
				}
				if operation.Done {
					break
				}
			}
			log.Printf("%s gae application created %s", appDeployment.Core.InstanceName, appToCreate.Id)
			// ffo.JSONMarshalIndentPrint(operation)
		} else {
			if strings.Contains(err.Error(), "403") {
				log.Printf("%s gae WARNING impossible to GET application %v", appDeployment.Core.InstanceName, err)
				return nil
			}
			return fmt.Errorf("gae appsService.Get(name) %v", err)
		}
	} else {
		log.Printf("%s gae application found %s", appDeployment.Core.InstanceName, app.Name)
	}
	return nil
}
