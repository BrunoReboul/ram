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

const lastAssetsQuery = `
SELECT
    assets.*
FROM
    (
        SELECT
            name,
            MAX(timestamp) AS timestamp
        FROM
            <assets>
        WHERE
            DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL <intervalDays> DAY)
            OR _PARTITIONTIME IS NULL
        GROUP BY
            name
        ORDER BY
            name
    ) AS latest_assets
    INNER JOIN (
        SELECT
            timestamp,
            name,
            owner,
            violationResolver,
            ancestryPathDisplayName,
            ancestryPath,
            ancestorsDisplayName,
            ancestors,
            assetType,
            deleted,
            projectID
        FROM
            <assets>
        WHERE
            DATE(_PARTITIONTIME) > DATE_SUB(CURRENT_DATE(), INTERVAL <intervalDays> DAY)
            OR _PARTITIONTIME IS NULL
    ) AS assets ON assets.name = latest_assets.name
    AND assets.timestamp = latest_assets.timestamp
`

func getLastAssetsQuery(projectID string, datasetName string, intervalDays int64) (query string) {
	assetsTableName := fmt.Sprintf("`%s.%s.assets`", projectID, datasetName)
	query = strings.Replace(lastAssetsQuery, "<assets>", assetsTableName, -1)
	query = strings.Replace(query, "<intervalDays>", fmt.Sprintf("%d", intervalDays), -1)
	return query
}
