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

import "cloud.google.com/go/bigquery"

// GetComplianceStatusSchema defines complianceStatus table schema
func GetComplianceStatusSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "assetName", Required: true, Type: bigquery.StringFieldType},
		{Name: "assetInventoryTimeStamp", Required: true, Type: bigquery.TimestampFieldType, Description: "When the asset change was captured"},
		{Name: "assetInventoryOrigin", Required: false, Type: bigquery.StringFieldType, Description: "Mean to capture the asset change: real-time or batch-export"},
		{Name: "ruleName", Required: true, Type: bigquery.StringFieldType},
		{Name: "ruleDeploymentTimeStamp", Required: true, Type: bigquery.TimestampFieldType, Description: "When the rule was assessed"},
		{Name: "compliant", Required: true, Type: bigquery.BooleanFieldType},
		{Name: "deleted", Required: true, Type: bigquery.BooleanFieldType},
	}
}
