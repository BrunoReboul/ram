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

func getDataset(ctx context.Context, datasetName string, location string, bigQueryClient *bigquery.Client) (dataset *bigquery.Dataset, err error) {
	dataset = bigQueryClient.Dataset(datasetName)
	datasetMetadata, err := dataset.Metadata(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var datasetToCreateMetadata bigquery.DatasetMetadata
			datasetToCreateMetadata.Name = datasetName
			datasetToCreateMetadata.Location = location
			datasetToCreateMetadata.Description = "Real-time Asset Monitor"
			datasetToCreateMetadata.Labels = map[string]string{"name": strings.ToLower(datasetName)}

			err = dataset.Create(ctx, &datasetToCreateMetadata)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					datasetMetadata, err = dataset.Metadata(ctx)
					if err != nil {
						return nil, err
					}
				}
				return nil, fmt.Errorf("dataset.Create %v", err)
			}
			log.Printf("Created dataset %s", datasetName)
			return dataset, nil
		}
	}
	needToUpdate := false
	if datasetMetadata.Labels != nil {
		if value, ok := datasetMetadata.Labels["name"]; ok {
			if value != datasetMetadata.Name {
				needToUpdate = true
			}
		} else {
			needToUpdate = true
		}
	} else {
		needToUpdate = true
	}
	if needToUpdate {
		var datasetMetadataToUpdate bigquery.DatasetMetadataToUpdate
		datasetMetadataToUpdate.SetLabel("name", strings.ToLower(datasetName))
		datasetMetadata, err = dataset.Update(ctx, datasetMetadataToUpdate, "")
		if err != nil {
			return nil, fmt.Errorf("ERROR when updating dataset labels %v", err)
		}
		log.Printf("Update dataset labels %s", datasetName)
	}
	return dataset, nil
}
