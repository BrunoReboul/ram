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

// GetViolationsSchema defines violations table schema
func GetViolationsSchema() bigquery.Schema {
	return bigquery.Schema{
		{
			Name:        "nonCompliance",
			Type:        bigquery.RecordFieldType,
			Description: "The violation information, aka why it is not compliant",
			Schema: bigquery.Schema{
				{Name: "message", Required: true, Type: bigquery.StringFieldType},
				{Name: "metadata", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{
			Name:        "functionConfig",
			Type:        bigquery.RecordFieldType,
			Description: "The settings of the cloud function hosting the rule check",
			Schema: bigquery.Schema{
				{Name: "functionName", Required: true, Type: bigquery.StringFieldType},
				{Name: "deploymentTime", Required: true, Type: bigquery.TimestampFieldType},
				{Name: "projectID", Required: false, Type: bigquery.StringFieldType},
				{Name: "environment", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{
			Name:        "constraintConfig",
			Type:        bigquery.RecordFieldType,
			Description: "The settings of the constraint used in conjonction with the rego template to assess the rule",
			Schema: bigquery.Schema{
				{Name: "kind", Required: false, Type: bigquery.StringFieldType},
				{
					Name: "metadata",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "name", Required: false, Type: bigquery.StringFieldType},
						{Name: "annotation", Required: false, Type: bigquery.StringFieldType},
					},
				},
				{
					Name: "spec",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "severity", Required: false, Type: bigquery.StringFieldType},
						{Name: "match", Required: false, Type: bigquery.StringFieldType},
						{Name: "parameters", Required: false, Type: bigquery.StringFieldType},
					},
				},
			},
		},
		{
			Name:        "feedMessage",
			Type:        bigquery.RecordFieldType,
			Description: "The message from Cloud Asset Inventory in realtime or from split dump in batch",
			Schema: bigquery.Schema{
				{
					Name: "asset",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "name", Required: true, Type: bigquery.StringFieldType},
						{Name: "owner", Required: false, Type: bigquery.StringFieldType},
						{Name: "violationResolver", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestryPathDisplayName", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestryPath", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestorsDisplayName", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestors", Required: false, Type: bigquery.StringFieldType},
						{Name: "assetType", Required: true, Type: bigquery.StringFieldType},
						{Name: "iamPolicy", Required: false, Type: bigquery.StringFieldType},
						{Name: "resource", Required: false, Type: bigquery.StringFieldType},
					},
				},
				{
					Name: "window",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "startTime", Required: true, Type: bigquery.TimestampFieldType},
					},
				},
				{Name: "origin", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{Name: "regoModules", Required: false, Type: bigquery.StringFieldType, Description: "The rego code, including the rule template used to assess the rule as a JSON document"},
	}
}
