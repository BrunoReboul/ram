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
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/bigquery"
)

func createUpdateView(ctx context.Context, tableName string, dataset *bigquery.Dataset, intervalDays int64) (err error) {
	var viewName, query string
	switch tableName {
	case "complianceStatus":
		viewName = "last_compliancestatus"
		query = getLastComplianceStatusQuery(dataset.ProjectID, dataset.DatasetID, intervalDays)
	case "violations":
		viewName = "active_violations"
		query = getActiveViolationsQuery(dataset.ProjectID, dataset.DatasetID, intervalDays)
	case "assets":
		viewName = "last_assets"
		query = getLastAssetsQuery(dataset.ProjectID, dataset.DatasetID, intervalDays)
	}
	table := dataset.Table(viewName)
	tableMetadataRetreived, err := table.Metadata(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var tableMetadata bigquery.TableMetadata
			tableMetadata.Name = viewName
			tableMetadata.Description = fmt.Sprintf("Real-time Asset Monitor - %s", viewName)
			tableMetadata.Labels = map[string]string{"name": strings.ToLower(viewName)}
			tableMetadata.ViewQuery = query
			tableMetadata.UseLegacySQL = false
			err = table.Create(ctx, &tableMetadata)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					return nil
				}
				return fmt.Errorf("create view %v", err)
			}
			log.Printf("Created view %s", viewName)
			return nil
		}
	}
	log.Printf("Found view %s", tableMetadataRetreived.Name)
	needToUpdate := false
	if tableMetadataRetreived.Labels != nil {
		if value, ok := tableMetadataRetreived.Labels["name"]; ok {
			if value != tableMetadataRetreived.Name {
				needToUpdate = true
			}
		} else {
			needToUpdate = true
		}
	} else {
		needToUpdate = true
	}
	if tableMetadataRetreived.Description != fmt.Sprintf("Real-time Asset Monitor - %s", viewName) {
		needToUpdate = true
	}
	if tableMetadataRetreived.ViewQuery != query {
		needToUpdate = true
	}
	if tableMetadataRetreived.UseLegacySQL {
		needToUpdate = true
	}
	if needToUpdate {
		var tableMetadataToUpdate bigquery.TableMetadataToUpdate
		tableMetadataToUpdate.SetLabel("name", strings.ToLower(viewName))
		tableMetadataToUpdate.Description = fmt.Sprintf("Real-time Asset Monitor - %s", viewName)
		tableMetadataToUpdate.ViewQuery = query
		tableMetadataToUpdate.UseLegacySQL = false
		tableMetadataRetreived, err = table.Update(ctx, tableMetadataToUpdate, "")
		if err != nil {
			return fmt.Errorf("ERROR when updating view %s %v", viewName, err)
		}
		log.Printf("View updated %s", tableMetadataRetreived.Name)
	}
	return nil
}
