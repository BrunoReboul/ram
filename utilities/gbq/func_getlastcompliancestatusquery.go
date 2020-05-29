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

const lastComplianceStatusQuery = `
SELECT
  complianceStates.ruleName,
  complianceStates.serviceName,
  REPLACE(
    complianceStates.ruleName,
    CONCAT("monitor_", complianceStates.serviceName, "_"),
    ""
  ) AS ruleNameShort,
  complianceStates.ruleDeploymentTimeStamp,
  complianceStates.compliant,
  NOT complianceStates.compliant AS notCompliant,
  complianceStates.assetName,
  complianceStates.assetInventoryTimeStamp,
  assets.owner,
  assets.violationResolver,
  assets.ancestryPathDisplayName,
  assets.ancestryPath,
  assets.ancestorsDisplayName,
  assets.ancestors,
  assets.assetType,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(0)] AS level0,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(1)] AS level1,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(2)] AS level2,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(3)] AS level3,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(4)] AS level4,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(5)] AS level5,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(6)] AS level6,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(7)] AS level7,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(8)] AS level8,
  SPLIT(assets.ancestryPathDisplayName, "/") [SAFE_OFFSET(9)] AS level9
FROM
  (
    SELECT
      status_for_latest_rules.ruleName,
      SPLIT(
        REPLACE(status_for_latest_rules.ruleName, "monitor_", ""),
        "_"
      ) [SAFE_OFFSET(0)] AS serviceName,
      status_for_latest_rules.ruleDeploymentTimeStamp,
      status_for_latest_rules.compliant,
      status_for_latest_rules.assetName,
      status_for_latest_rules.assetInventoryTimeStamp,
    FROM
      (
        SELECT
          assetName,
          MAX(assetInventoryTimeStamp) AS assetInventoryTimeStamp
        FROM
          <complianceStatus>
        WHERE
          DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL 1 YEAR)
          OR _PARTITIONTIME IS NULL
        GROUP BY
          assetName
        ORDER BY
          assetName
      ) AS latest_asset_inventory
      INNER JOIN (
        SELECT
          complianceStatus.*
        FROM
          (
            SELECT
              ruleName,
              MAX(ruleDeploymentTimeStamp) AS ruleDeploymentTimeStamp
            FROM
              <complianceStatus>
            WHERE
              DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL 1 YEAR)
              OR _PARTITIONTIME IS NULL
            GROUP BY
              ruleName
            ORDER BY
              ruleName
          ) AS latest_rules
          INNER JOIN (
            SELECT
              *
            FROM
              <complianceStatus>
            WHERE
              DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL 1 YEAR)
              OR _PARTITIONTIME IS NULL
          ) AS complianceStatus ON complianceStatus.ruleName = latest_rules.ruleName
          AND complianceStatus.ruleDeploymentTimeStamp = latest_rules.ruleDeploymentTimeStamp
      ) AS status_for_latest_rules ON status_for_latest_rules.assetName = latest_asset_inventory.assetName
      AND status_for_latest_rules.assetInventoryTimeStamp = latest_asset_inventory.assetInventoryTimeStamp
    WHERE
      status_for_latest_rules.deleted = FALSE
  ) AS complianceStates
  LEFT JOIN (
    SELECT
      name,
      owner,
      violationResolver,
      ancestryPathDisplayName,
      ancestryPath,
      ancestorsDisplayName,
      ancestors,
      assetType
    FROM
      <last_assets>
  ) AS assets ON complianceStates.assetName = assets.name
ORDER BY
  complianceStates.ruleName,
  complianceStates.ruleDeploymentTimeStamp,
  complianceStates.compliant,
  complianceStates.assetName,
  complianceStates.assetInventoryTimeStamp
`

func getLastComplianceStatusQuery(projectID string, datasetName string) (query string) {
	lastAssetsViewName := fmt.Sprintf("`%s.%s.last_assets`", projectID, datasetName)
	query = strings.Replace(lastComplianceStatusQuery, "<last_assets>", lastAssetsViewName, -1)
	complianceStatusTableName := fmt.Sprintf("`%s.%s.complianceStatus`", projectID, datasetName)
	query = strings.Replace(query, "<complianceStatus>", complianceStatusTableName, -1)
	return query
}
