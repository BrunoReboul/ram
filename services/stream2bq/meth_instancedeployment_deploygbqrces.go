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

package stream2bq

import (
	"fmt"

	"github.com/BrunoReboul/ram/utilities/gbq"
)

func (instanceDeployment *InstanceDeployment) deployGBQRces() (err error) {
	var tableNameList = []string{"complianceStatus", "violations", "assets"}
	tableName := instanceDeployment.Settings.Instance.Bigquery.TableName
	datasetLocation := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Location
	datasetName := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Name
	intervalDays := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Views.IntervalDays
	if intervalDays == 0 {
		intervalDays = 365
	}

	switch tableName {
	case "complianceStatus":
		_, err = gbq.GetComplianceStatus(instanceDeployment.Core.Ctx, instanceDeployment.Core.Services.BigqueryClient, datasetLocation, datasetName, intervalDays)
		if err != nil {
			return fmt.Errorf("gbq.GetComplianceStatus %v", err)
		}
	case "violations":
		_, err = gbq.GetViolations(instanceDeployment.Core.Ctx, instanceDeployment.Core.Services.BigqueryClient, datasetLocation, datasetName, intervalDays)
		if err != nil {
			return fmt.Errorf("gbq.GetViolations %v", err)
		}
	case "assets":
		_, err = gbq.GetAssets(instanceDeployment.Core.Ctx, instanceDeployment.Core.Services.BigqueryClient, datasetLocation, datasetName, intervalDays)
		if err != nil {
			return fmt.Errorf("gbq.GetAssets %v", err)
		}
	default:
		return fmt.Errorf("Unsupported tablename %s supported are %v", tableName, tableNameList)
	}
	return nil
}
