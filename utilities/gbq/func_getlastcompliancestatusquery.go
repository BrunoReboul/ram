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
WITH complianceStatus0 AS (
    SELECT
      *
    FROM
      <complianceStatus>
    WHERE
      DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL <intervalDays> DAY)
      OR _PARTITIONTIME IS NULL
  ),
  assets AS (
    SELECT
      name,
      owner,
      violationResolver,
      ancestryPathDisplayName,
      ancestryPath,
      ancestorsDisplayName,
      ancestors,
      assetType,
      projectID
    FROM
      <last_assets>
),
  latest_asset_inventory_per_rule AS (
    SELECT
      assetName,
      ruleName,
      MAX(assetInventoryTimeStamp) AS assetInventoryTimeStamp
    FROM
      complianceStatus0
    GROUP BY
      assetName,
      ruleName
    ORDER BY
      assetName,
      ruleName
  ),
  latest_rules AS (
    SELECT
      ruleName,
      MAX(ruleDeploymentTimeStamp) AS ruleDeploymentTimeStamp
    FROM
      complianceStatus0
    GROUP BY
      ruleName
    ORDER BY
      ruleName
  ),
  status_for_latest_rules AS (
    SELECT
      complianceStatus0.*
    FROM
      latest_rules
      INNER JOIN complianceStatus0 ON complianceStatus0.ruleName = latest_rules.ruleName
      AND complianceStatus0.ruleDeploymentTimeStamp = latest_rules.ruleDeploymentTimeStamp
  ),
  complianceStatus1 AS (
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
      IF(
        SPLIT(status_for_latest_rules.assetName, "/") [SAFE_OFFSET(2)] = "directories",
        CONCAT(
          SPLIT(status_for_latest_rules.assetName, "/") [SAFE_OFFSET(2)],
          "/",
          SPLIT(status_for_latest_rules.assetName, "/") [SAFE_OFFSET(3)]
        ),
        NULL
      ) AS directoryPath,
      IF(
        SPLIT(status_for_latest_rules.assetName, "/") [SAFE_OFFSET(2)] = "directories",
        CASE
          SPLIT(status_for_latest_rules.assetName, "/") [SAFE_OFFSET(6)]
          WHEN "members" THEN "www.googleapis.com/admin/directory/members"
          WHEN "groupSettings" THEN "groupssettings.googleapis.com/groupSettings"
          ELSE NULL
        END,
        NULL
      ) AS directoryAssetType,
    FROM
      latest_asset_inventory_per_rule
      INNER JOIN status_for_latest_rules ON status_for_latest_rules.assetName = latest_asset_inventory_per_rule.assetName
      AND status_for_latest_rules.ruleName = latest_asset_inventory_per_rule.ruleName
      AND status_for_latest_rules.assetInventoryTimeStamp = latest_asset_inventory_per_rule.assetInventoryTimeStamp
    WHERE
      status_for_latest_rules.deleted = FALSE
  ),
  complianceStatus AS (
    SELECT
      complianceStatus1.ruleName,
      complianceStatus1.serviceName,
      REPLACE(
        complianceStatus1.ruleName,
        CONCAT("monitor_", complianceStatus1.serviceName, "_"),
        ""
      ) AS ruleNameShort,
      complianceStatus1.ruleDeploymentTimeStamp,
      complianceStatus1.compliant,
      NOT complianceStatus1.compliant AS notCompliant,
      complianceStatus1.assetName,
      complianceStatus1.assetInventoryTimeStamp,
      assets.owner,
      assets.violationResolver,
      IFNULL(
        assets.ancestryPath,
        complianceStatus1.directoryPath
      ) AS ancestryPath,
      IFNULL(
        assets.ancestryPathDisplayName,
        IFNULL(
          assets.ancestryPath,
          complianceStatus1.directoryPath
        )
      ) AS ancestryPathDisplayName,
      IF(
        ARRAY_LENGTH(assets.ancestorsDisplayName) > 0,
        assets.ancestorsDisplayName,
        assets.ancestors
      ) AS ancestorsDisplayName,
      assets.ancestors,
      IFNULL(
        assets.assetType,
        complianceStatus1.directoryAssetType
      ) AS assetType,
      assets.projectID,
    FROM
      complianceStatus1
      LEFT JOIN assets ON complianceStatus1.assetName = assets.name
  )
  SELECT
    complianceStatus.*,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(0)] AS level0,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(1)] AS level1,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(2)] AS level2,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(3)] AS level3,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(4)] AS level4,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(5)] AS level5,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(6)] AS level6,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(7)] AS level7,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(8)] AS level8,
    SPLIT(complianceStatus.ancestryPathDisplayName, "/") [SAFE_OFFSET(9)] AS level9
  FROM
    complianceStatus
  ORDER BY
    complianceStatus.ruleName,
    complianceStatus.ruleDeploymentTimeStamp,
    complianceStatus.compliant,
    complianceStatus.assetName,
    complianceStatus.assetInventoryTimeStamp
`

func getLastComplianceStatusQuery(projectID string, datasetName string, intervalDays int64) (query string) {
	lastAssetsViewName := fmt.Sprintf("`%s.%s.last_assets`", projectID, datasetName)
	query = strings.Replace(lastComplianceStatusQuery, "<last_assets>", lastAssetsViewName, -1)
	complianceStatusTableName := fmt.Sprintf("`%s.%s.complianceStatus`", projectID, datasetName)
	query = strings.Replace(query, "<complianceStatus>", complianceStatusTableName, -1)
	query = strings.Replace(query, "<intervalDays>", fmt.Sprintf("%d", intervalDays), -1)
	return query
}
