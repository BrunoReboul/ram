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
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
)

func getTable(ctx context.Context, tableName string, dataset *bigquery.Dataset) (table *bigquery.Table, err error) {
	var schema bigquery.Schema
	switch tableName {
	case "complianceStatus":
		schema = GetComplianceStatusSchema()
	case "violations":
		schema = GetViolationsSchema()
	case "assets":
		schema = GetAssetsSchema()
	}

	table = dataset.Table(tableName)
	tableMetadata, err := table.Metadata(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var tableToCreateMetadata bigquery.TableMetadata
			tableToCreateMetadata.Name = tableName
			tableToCreateMetadata.Description = fmt.Sprintf("Real-time Asset Monitor - %s", tableName)
			tableToCreateMetadata.Labels = map[string]string{"name": strings.ToLower(tableName)}

			var timePartitioning bigquery.TimePartitioning
			timePartitioning.Type = "DAY"
			timePartitioning.Expiration = time.Duration(0)
			tableToCreateMetadata.TimePartitioning = &timePartitioning
			tableToCreateMetadata.Schema = schema

			err = table.Create(ctx, &tableToCreateMetadata)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					tableMetadata, err = table.Metadata(ctx)
					if err != nil {
						return nil, err
					}
				}
				return nil, fmt.Errorf("table.Create %v", err)
			}
			log.Printf("gbq created table %s", tableName)
			return table, nil
		}
		return nil, fmt.Errorf("table.Metadata(ctx) %v", err)
	}
	log.Printf("gbq found table %s", tableName)
	needToUpdate := false
	var tableMetadataToUpdate bigquery.TableMetadataToUpdate
	// Labels
	if tableMetadata.Labels != nil {
		if value, ok := tableMetadata.Labels["name"]; ok {
			if strings.ToLower(value) != strings.ToLower(tableMetadata.Name) {
				log.Printf("gbq found %s wants %s for label 'name' on table %s", value, tableMetadata.Name, tableName)
				needToUpdate = true
			}
		} else {
			log.Printf("gbq label 'name' not found on table %s", tableName)
			needToUpdate = true
		}
	} else {
		log.Printf("gbq no label found on table %s", tableName)
		needToUpdate = true
	}
	if needToUpdate {
		tableMetadataToUpdate.SetLabel("name", strings.ToLower(tableName))
		log.Printf("gbq need to update table labels %s", tableName)
	}
	// Table partitioning
	if tableMetadata.TimePartitioning.Expiration != time.Duration(0) {
		var timePartitioning bigquery.TimePartitioning
		timePartitioning.Expiration = time.Duration(0)
		timePartitioning.Type = tableMetadata.TimePartitioning.Type

		tableMetadataToUpdate.TimePartitioning = &timePartitioning
		log.Printf("gbq need to update partition expiration on table %s", tableName)
		needToUpdate = true
	}
	if reflect.DeepEqual(tableMetadata.Schema, schema) {
		tableMetadataToUpdate.Schema = schema
		log.Printf("gbq need to update schema on table %s", tableName)
		needToUpdate = true
	}
	// Update
	if needToUpdate {
		tableMetadata, err = table.Update(ctx, tableMetadataToUpdate, "")
		if err != nil {
			return nil, fmt.Errorf("ERROR when updating table labels %v", err)
		}
		log.Printf("gbq table updated %s", tableName)
	}
	return table, nil
}
