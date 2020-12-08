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
	"io/ioutil"
	"log"

	"github.com/BrunoReboul/ram/utilities/str"
	"google.golang.org/api/iam/v1"
)

// getKeyJSONdataAndCleanKeys get the service account key to build a JWT and clean older keys
func getKeyJSONdataAndCleanKeys(ctx context.Context,
	serviceAccountEmail string,
	keyJSONFilePath string,
	projectID string,
	serviceAccountKeyNames []string,
	logEntryPrefix string) (
	keyRestAPIFormat keyRestAPIFormat,
	err error) {
	var keyJSONdata []byte
	var currentKeyName string
	var iamService *iam.Service

	iamService, err = iam.NewService(ctx)
	if err != nil {
		log.Printf("%s ERROR - iam.NewService: %v", logEntryPrefix, err)
		return keyRestAPIFormat, err
	}
	resource := "projects/-/serviceAccounts/" + serviceAccountEmail
	listServiceAccountKeyResponse, err := iamService.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		log.Printf("%s ERROR - iamService.Projects.ServiceAccounts.Keys.List: %v", logEntryPrefix, err)
		return keyRestAPIFormat, err
	}
	keyJSONdata, err = ioutil.ReadFile(keyJSONFilePath)
	if err != nil {
		log.Printf("%s ERROR - ioutil.ReadFile(keyJSONFilePath): %v", logEntryPrefix, err)
		return keyRestAPIFormat, err
	}
	err = json.Unmarshal(keyJSONdata, &keyRestAPIFormat)
	if err != nil {
		log.Printf("%s ERROR - json.Unmarshal(keyJSONdata, &keyRestAPIFormat): %v", logEntryPrefix, err)
		return keyRestAPIFormat, err
	}
	currentKeyName = keyRestAPIFormat.Name

	// Clean keys
	for _, serviceAccountKey := range listServiceAccountKeyResponse.Keys {
		if serviceAccountKey.Name == currentKeyName {
			log.Printf("%s keep current key ValidAfterTime %s named %s", logEntryPrefix, serviceAccountKey.ValidAfterTime, serviceAccountKey.Name)
		} else {
			if str.Find(serviceAccountKeyNames, serviceAccountKey.Name) {
				log.Printf("%s keep recorded key ValidAfterTime %s named %s", logEntryPrefix, serviceAccountKey.ValidAfterTime, serviceAccountKey.Name)
			} else {
				if serviceAccountKey.KeyType == "SYSTEM_MANAGED" {
					log.Printf("%s ignore SYSTEM_MANAGED key named %s", logEntryPrefix, serviceAccountKey.Name)
				} else {
					log.Printf("%s delete KeyType %s ValidAfterTime %s key name %s", logEntryPrefix, serviceAccountKey.KeyType, serviceAccountKey.ValidAfterTime, serviceAccountKey.Name)
					_, err = iamService.Projects.ServiceAccounts.Keys.Delete(serviceAccountKey.Name).Do()
					if err != nil {
						log.Printf("%s ERROR - iamService.Projects.ServiceAccounts.Keys.Delete: %v", logEntryPrefix, err)
						return keyRestAPIFormat, err
					}
				}
			}
		}
	}
	return keyRestAPIFormat, nil
}
