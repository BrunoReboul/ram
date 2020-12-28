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

package cai

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// BuildAncestorsDisplayName build a slice of Ancestor friendly name from a slice of ancestors
func BuildAncestorsDisplayName(ctx context.Context,
	ancestors []string,
	collectionID string,
	firestoreClient *firestore.Client,
	cloudresourcemanagerService *cloudresourcemanager.Service,
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) (ancestorsDisplayName []string, projectID string) {
	cnt := len(ancestors)
	ancestorsDisplayName = make([]string, len(ancestors))
	for idx := 0; idx < cnt; idx++ {
		displayName, p := getDisplayName(ctx,
			ancestors[idx], collectionID, firestoreClient, cloudresourcemanagerService, cloudresourcemanagerServiceV2)
		ancestorsDisplayName[idx] = displayName
		if p != "" {
			projectID = p
		}
	}
	return ancestorsDisplayName, projectID
}
