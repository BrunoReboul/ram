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

	"cloud.google.com/go/firestore"
)

// ListKeyNames retrieve the list of recorded keyname for a given microservice
func ListKeyNames(ctx context.Context, firestoreClient *firestore.Client, serviceName string) (serviceAccountKeyNames []string, err error) {
	docRefs, err := firestoreClient.Collection(serviceName).DocumentRefs(ctx).GetAll()
	if err != nil {
		return serviceAccountKeyNames, err
	}
	for _, docRef := range docRefs {
		docSnap, err := docRef.Get(ctx)
		if err != nil {
			return serviceAccountKeyNames, err
		}
		document := docSnap.Data()
		var serviceAccountKeyNameInterface interface{} = document["serviceAccountKeyName"]
		if serviceAccountKeyName, ok := serviceAccountKeyNameInterface.(string); ok {
			serviceAccountKeyNames = append(serviceAccountKeyNames, serviceAccountKeyName)
		}
	}
	return serviceAccountKeyNames, nil
}
