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
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/BrunoReboul/ram/utilities/glo"
)

// RecordDump record a dump stepStack in firestore
func RecordDump(ctx context.Context,
	dumpNameFull string,
	firestoreClient *firestore.Client,
	stepStack glo.Steps,
	microserviceName string,
	instanceName string,
	environment string,
	pubSubID string,
	retriesNumber time.Duration) (err error) {
	var i time.Duration
	var dumpName string
	if strings.Contains(dumpNameFull, "/") {
		parts := strings.Split(dumpNameFull, "/")
		dumpName = strings.Replace(parts[1], ".dump", "", 1)
	} else {
		dumpName = strings.Replace(dumpNameFull, ".dump", "", 1)
	}

	documentPath := fmt.Sprintf("dumps/%s", dumpName)

	for i = 0; i < retriesNumber; i++ {
		_, err = firestoreClient.Doc(documentPath).Get(ctx)
		if err != nil {
			if strings.Contains(strings.ToLower(strings.Replace(err.Error(), " ", "", -1)), "notfound") {
				_, err = firestoreClient.Doc(documentPath).Set(ctx, map[string]interface{}{
					"stepStack": stepStack,
				})
				if err != nil {
					log.Println(glo.Entry{
						MicroserviceName:   microserviceName,
						InstanceName:       instanceName,
						Environment:        environment,
						Severity:           "WARNING",
						Message:            "recordDump cannot set firestore doc",
						Description:        fmt.Sprintf("iteration %d firestoreClient.Doc(documentPath).Set %s %v", i, documentPath, err),
						TriggeringPubsubID: pubSubID,
					})
					time.Sleep(i * 100 * time.Millisecond)
				} else {
					log.Println(glo.Entry{
						MicroserviceName:   microserviceName,
						InstanceName:       instanceName,
						Environment:        environment,
						Severity:           "INFO",
						Message:            fmt.Sprintf("dump stepStack recorded %s", documentPath),
						TriggeringPubsubID: pubSubID,
					})
					return nil
				}
			} else {
				log.Println(glo.Entry{
					MicroserviceName:   microserviceName,
					InstanceName:       instanceName,
					Environment:        environment,
					Severity:           "WARNING",
					Message:            "recordDump cannot get firestore doc",
					Description:        fmt.Sprintf("iteration %d firestoreClient.Doc(documentPath).Get %s %v", i, documentPath, err),
					TriggeringPubsubID: pubSubID,
				})
				time.Sleep(i * 100 * time.Millisecond)
			}
		} else {
			_, err = firestoreClient.Doc(documentPath).Update(ctx, []firestore.Update{
				{
					Path:  "stepStack",
					Value: stepStack,
				},
			})
			if err != nil {
				log.Println(glo.Entry{
					MicroserviceName:   microserviceName,
					InstanceName:       instanceName,
					Environment:        environment,
					Severity:           "WARNING",
					Message:            "recordDump cannot update firestore doc",
					Description:        fmt.Sprintf("iteration %d firestoreClient.Doc(documentPath).Update %s %v", i, documentPath, err),
					TriggeringPubsubID: pubSubID,
				})
				time.Sleep(i * 100 * time.Millisecond)
			} else {
				log.Println(glo.Entry{
					MicroserviceName:   microserviceName,
					InstanceName:       instanceName,
					Environment:        environment,
					Severity:           "INFO",
					Message:            fmt.Sprintf("dump stepStack updated %s", documentPath),
					TriggeringPubsubID: pubSubID,
				})
				return nil
			}
		}
	}
	return err
}
