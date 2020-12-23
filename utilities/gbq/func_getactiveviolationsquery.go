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
	"fmt"
	"strings"
)

const activeViolationsQuery = `
SELECT
    violations.*,
    compliancestatus.serviceName,
    compliancestatus.ruleNameShort,
    compliancestatus.level0,
    compliancestatus.level1,
    compliancestatus.level2,
    compliancestatus.level3,
    compliancestatus.level4,
    compliancestatus.level5,
    compliancestatus.level6,
    compliancestatus.level7,
    compliancestatus.level8,
    compliancestatus.level9
FROM
    <last_compliancestatus> AS compliancestatus
    INNER JOIN (
        SELECT
            *
        FROM
          <violations>
        WHERE
            DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL <intervalDays> DAY)
            OR _PARTITIONTIME IS NULL
    ) AS violations ON violations.functionConfig.functionName = compliancestatus.ruleName
    AND violations.functionConfig.deploymentTime = compliancestatus.ruleDeploymentTimeStamp
    AND violations.feedMessage.asset.name = compliancestatus.assetName
    AND violations.feedMessage.window.startTime = compliancestatus.assetInventoryTimeStamp
`

func getActiveViolationsQuery(projectID string, datasetName string, intervalDays int64) (query string) {
	lastComplianceStatusViewName := fmt.Sprintf("`%s.%s.last_compliancestatus`", projectID, datasetName)
	query = strings.Replace(activeViolationsQuery, "<last_compliancestatus>", lastComplianceStatusViewName, -1)
	violationsTableName := fmt.Sprintf("`%s.%s.violations`", projectID, datasetName)
	query = strings.Replace(query, "<violations>", violationsTableName, -1)
	query = strings.Replace(query, "<intervalDays>", fmt.Sprintf("%d", intervalDays), -1)
	return query
}
