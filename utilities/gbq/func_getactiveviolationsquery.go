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
  <last_compliancestatus>.serviceName,
  <last_compliancestatus>.ruleNameShort,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(0)] AS level0,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(1)] AS level1,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(2)] AS level2,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(3)] AS level3,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(4)] AS level4,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(5)] AS level5,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(6)] AS level6,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(7)] AS level7,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(8)] AS level8,
  SPLIT(feedmessage.asset.ancestryPathDisplayName, "/") [SAFE_OFFSET(9)] AS level9
FROM
  <last_compliancestatus>
  INNER JOIN (
    SELECT
      *
    FROM
      <violations>
    WHERE
      DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL <intervalDays> DAY)
      OR _PARTITIONTIME IS NULL
  ) AS violations ON violations.functionConfig.functionName = <last_compliancestatus>.ruleName
  AND violations.functionConfig.deploymentTime = <last_compliancestatus>.ruleDeploymentTimeStamp
  AND violations.feedMessage.asset.name = <last_compliancestatus>.assetName
  AND violations.feedMessage.window.startTime = <last_compliancestatus>.assetInventoryTimeStamp
`

func getActiveViolationsQuery(projectID string, datasetName string, intervalDays int64) (query string) {
	lastComplianceStatusViewName := fmt.Sprintf("`%s.%s.last_compliancestatus`", projectID, datasetName)
	query = strings.Replace(activeViolationsQuery, "<last_compliancestatus>", lastComplianceStatusViewName, -1)
	violationsTableName := fmt.Sprintf("`%s.%s.violations`", projectID, datasetName)
	query = strings.Replace(query, "<violations>", violationsTableName, -1)
	query = strings.Replace(query, "<intervalDays>", fmt.Sprintf("%d", intervalDays), -1)
	return query
}
