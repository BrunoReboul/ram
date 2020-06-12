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

package gbq

import (
	"context"

	"cloud.google.com/go/bigquery"
)

// GetComplianceStatus provision compliancestatus table, view, and dependencies
func GetComplianceStatus(ctx context.Context, bigQueryClient *bigquery.Client, location string, datasetName string) (table *bigquery.Table, err error) {
	dataset, err := getDataset(ctx, datasetName, location, bigQueryClient)
	if err != nil {
		return nil, err
	}
	complianceStatusTable, err := getTable(ctx, "complianceStatus", dataset)
	if err != nil {
		return nil, err
	}
	// Ensure assets table and view exist
	_, err = GetAssets(ctx, bigQueryClient, location, datasetName)
	if err != nil {
		return nil, err
	}
	err = createUpdateView(ctx, "complianceStatus", dataset)
	if err != nil {
		return nil, err
	}
	return complianceStatusTable, nil
}
