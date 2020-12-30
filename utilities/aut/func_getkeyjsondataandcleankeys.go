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

package aut

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/BrunoReboul/ram/utilities/glo"
	"github.com/BrunoReboul/ram/utilities/str"
	"google.golang.org/api/iam/v1"
)

// getKeyJSONdataAndCleanKeys get the service account key to build a JWT and clean older keys
func getKeyJSONdataAndCleanKeys(ctx context.Context,
	serviceAccountEmail string,
	keyJSONFilePath string,
	projectID string,
	serviceAccountKeyNames []string,
	initID string,
	microserviceName string,
	instanceName string,
	environment string) (
	keyRestAPIFormat keyRestAPIFormat,
	err error) {
	var keyJSONdata []byte
	var currentKeyName string
	var iamService *iam.Service

	iamService, err = iam.NewService(ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: microserviceName,
			InstanceName:     instanceName,
			Environment:      environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("iam.NewService %v", err),
			InitID:           initID,
		})
		return keyRestAPIFormat, err
	}
	resource := "projects/-/serviceAccounts/" + serviceAccountEmail
	listServiceAccountKeyResponse, err := iamService.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: microserviceName,
			InstanceName:     instanceName,
			Environment:      environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("iamService.Projects.ServiceAccounts.Keys.List %v", err),
			InitID:           initID,
		})
		return keyRestAPIFormat, err
	}
	keyJSONdata, err = ioutil.ReadFile(keyJSONFilePath)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: microserviceName,
			InstanceName:     instanceName,
			Environment:      environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("ioutil.ReadFile(keyJSONFilePath) %v", err),
			InitID:           initID,
		})
		return keyRestAPIFormat, err
	}
	err = json.Unmarshal(keyJSONdata, &keyRestAPIFormat)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: microserviceName,
			InstanceName:     instanceName,
			Environment:      environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("json.Unmarshal(keyJSONdata, &keyRestAPIFormat) %v", err),
			InitID:           initID,
		})
		return keyRestAPIFormat, err
	}
	currentKeyName = keyRestAPIFormat.Name

	// Clean keys
	for _, serviceAccountKey := range listServiceAccountKeyResponse.Keys {
		if serviceAccountKey.Name == currentKeyName {
			log.Println(glo.Entry{
				MicroserviceName: microserviceName,
				InstanceName:     instanceName,
				Environment:      environment,
				Severity:         "INFO",
				Message:          "init_keep_current_key",
				Description:      fmt.Sprintf("ValidAfterTime %s named %s", serviceAccountKey.ValidAfterTime, serviceAccountKey.Name),
				InitID:           initID,
			})
		} else {
			if str.Find(serviceAccountKeyNames, serviceAccountKey.Name) {
				log.Println(glo.Entry{
					MicroserviceName: microserviceName,
					InstanceName:     instanceName,
					Environment:      environment,
					Severity:         "INFO",
					Message:          "init_keep_recorded_key",
					Description:      fmt.Sprintf("ValidAfterTime %s named %s", serviceAccountKey.ValidAfterTime, serviceAccountKey.Name),
					InitID:           initID,
				})
			} else {
				if serviceAccountKey.KeyType == "SYSTEM_MANAGED" {
					log.Println(glo.Entry{
						MicroserviceName: microserviceName,
						InstanceName:     instanceName,
						Environment:      environment,
						Severity:         "INFO",
						Message:          "init_ignore_system_managed_key",
						Description:      fmt.Sprintf("named %s", serviceAccountKey.Name),
						InitID:           initID,
					})
				} else {
					log.Println(glo.Entry{
						MicroserviceName: microserviceName,
						InstanceName:     instanceName,
						Environment:      environment,
						Severity:         "INFO",
						Message:          "init_delete_key",
						Description:      fmt.Sprintf("ValidAfterTime %s named %s", serviceAccountKey.ValidAfterTime, serviceAccountKey.Name),
						InitID:           initID,
					})
					_, err = iamService.Projects.ServiceAccounts.Keys.Delete(serviceAccountKey.Name).Do()
					if err != nil {
						log.Println(glo.Entry{
							MicroserviceName: microserviceName,
							InstanceName:     instanceName,
							Environment:      environment,
							Severity:         "CRITICAL",
							Message:          "init_cannot_delete_key",
							Description:      fmt.Sprintf("iamService.Projects.ServiceAccounts.Keys.Delete: %v", err),
							InitID:           initID,
						})
						return keyRestAPIFormat, err
					}
				}
			}
		}
	}
	return keyRestAPIFormat, nil
}
