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

package gfs

import (
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/BrunoReboul/ram/utilities/deploy"
)

// RecordKeyName records the service account key name used by a deployed microservice instance
func RecordKeyName(core *deploy.Core, serviceAccountKeyName string, retriesNumber time.Duration) (err error) {
	var i time.Duration
	documentPath := fmt.Sprintf("%s/%s", core.ServiceName, core.InstanceName)
	for i = 0; i < retriesNumber; i++ {
		_, err = core.Services.FirestoreClient.Doc(documentPath).Get(core.Ctx)
		if err != nil {
			if strings.Contains(strings.ToLower(strings.Replace(err.Error(), " ", "", -1)), "notfound") {
				_, err = core.Services.FirestoreClient.Doc(documentPath).Set(core.Ctx, map[string]interface{}{
					"serviceAccountKeyName": serviceAccountKeyName,
				})
				if err != nil {
					log.Printf("ERROR - iteration %d core.Services.FirestoreClient.Doc(documentPath).Set %v", i, err)
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					return nil
				}
			} else {
				log.Printf("ERROR - iteration %d core.Services.FirestoreClient.Doc(documentPath).Get %v", i, err)
				time.Sleep(i * 100 * time.Millisecond)
			}
		} else {
			_, err = core.Services.FirestoreClient.Doc(documentPath).Update(core.Ctx, []firestore.Update{
				{
					Path:  "serviceAccountKeyName",
					Value: serviceAccountKeyName,
				},
			})
			if err != nil {
				log.Printf("ERROR - iteration %d core.Services.FirestoreClient.Doc(documentPath).Update %v", i, err)
				time.Sleep(i * 100 * time.Millisecond)
			} else {
				return nil
			}
		}
	}
	return err
}
