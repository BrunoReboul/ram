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

// GetDoc check if a document exist with retries
func GetDoc(ctx context.Context,
	firestoreClient *firestore.Client,
	documentPath string,
	retriesNumber time.Duration) (*firestore.DocumentSnapshot, bool) {
	var documentSnap *firestore.DocumentSnapshot
	var err error
	var i time.Duration
	for i = 0; i < retriesNumber; i++ {
		documentSnap, err = firestoreClient.Doc(documentPath).Get(ctx)
		if err != nil {
			// Retry are for transient, not for doc not found
			if strings.Contains(strings.ToLower(err.Error()), "notfound") {
				log.Println(glo.Entry{
					Severity: "WARNING",
					Message:  "no_found_in_cache",
				})
				return documentSnap, false
			}
			log.Println(glo.Entry{
				Severity:    "CRITICAL",
				Message:     "redo_on_transient",
				Description: fmt.Sprintf("iteration %d firestoreClient.Doc(documentPath).Get(ctx) %v", i, err),
			})
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			// Found
			return documentSnap, documentSnap.Exists()
		}
	}
	return documentSnap, false
}
