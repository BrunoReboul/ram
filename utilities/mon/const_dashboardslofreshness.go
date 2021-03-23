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

package mon

// SLOFreshnessTiles freshness SLO dashboard JSON template
const SLOFreshnessTiles = `
[
	{
		"height": 2,
		"width": 4,
		"widget": {
			"title": "<scope> <flow> <sloText> < <thresholdText>",
			"text": {
				"content": "**Freshness**: <sloText> of <scope> configurations from <flow> flow over the last 28 days should be analyzed in less than <thresholdText>.",
				"format": "MARKDOWN"
			}
		}
	},
	{
		"height": 2,
		"width": 3,
		"xPos": 4,
		"widget": {
			"title": "SLI vs SLO",
			"scorecard": {
				"gaugeView": {
					"lowerBound": <lowerBound>,
					"upperBound": 1.0
				},
				"thresholds": [
					{
						"color": "RED",
						"direction": "BELOW",
						"value": <slo>
					}
				],
				"timeSeriesQuery": {
					"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n| filter metric.microservice_name == 'stream2bq'\n| filter metric.origin == '<origin>'\n| align delta(28d)\n| every 28d\n| within 28d\n| group_by [metric.microservice_name]\n| fraction_less_than_from <thresholdSeconds>"
				}
			}
		}
	},
	{
		"height": 2,
		"width": 3,
		"xPos": 7,
		"widget": {
			"title": "Remaining ERROR BUDGET",
			"scorecard": {
				"thresholds": [
					{
						"color": "YELLOW",
						"direction": "BELOW",
						"value": 0.1
					}
				],
				"timeSeriesQuery": {
					"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n| filter metric.microservice_name == 'stream2bq'\n| filter metric.origin == '<origin>'\n| align delta(28d)\n| every 28d\n| within 28d\n| group_by [metric.microservice_name]\n| fraction_less_than_from <thresholdSeconds>\n| neg\n| add 1\n| div 0.01\n| neg\n| add 1"
				}
			}
		}
	},
	{
		"height": 2,
		"width": 2,
		"xPos": 10,
		"widget": {
			"title": "Configurations analyzed in 28 days",
			"scorecard": {
				"sparkChartView": {
					"sparkChartType": "SPARK_LINE"
				},
				"timeSeriesQuery": {
					"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n| filter metric.microservice_name == 'stream2bq'\n| filter metric.origin == '<origin>'\n| align delta(28d)\n| every 28d\n| within 28d\n| group_by [metric.microservice_name]\n| count_from"
				}
			}
		}
	},
	{
		"height": 9,
		"width": 3,
		"xPos": 9,
		"yPos": 2,
		"widget": {
			"title": "Last 28days heatmap",
			"xyChart": {
				"chartOptions": {
					"mode": "COLOR"
				},
				"dataSets": [
					{
						"plotType": "HEATMAP",
						"timeSeriesQuery": {
							"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n| filter (metric.microservice_name == 'stream2bq')\n| filter metric.origin == '<origin>'\n| align delta(28d)\n| every 28d\n| within 28d\n| group_by [metric.microservice_name]\n| graph_period 28d"
						}
					}
				],
				"timeshiftDuration": "0s",
				"yAxis": {
					"label": "y1Axis",
					"scale": "LINEAR"
				}
			}
		}
	},
	{
		"height": 3,
		"width": 9,
		"yPos": 2,
		"widget": {
			"title": "Error budget burnrate on 7d sliding windows - Email when > 1.5",
			"xyChart": {
				"chartOptions": {
					"mode": "COLOR"
				},
				"dataSets": [
					{
						"plotType": "LINE",
						"timeSeriesQuery": {
							"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n|filter metric.microservice_name == 'stream2bq'\n| filter metric.origin == '<origin>'\n| align delta(1m)\n| every 1m\n| group_by [metric.microservice_name], sliding(7d)\n| fraction_less_than_from <thresholdSeconds>\n| neg\n| add 1\n| div 0.01\n| cast_units \"1\""
						}
					}
				],
				"thresholds": [
					{
						"value": 1.5
					}
				],
				"timeshiftDuration": "0s",
				"yAxis": {
					"label": "y1Axis",
					"scale": "LINEAR"
				}
			}
		}
	},
	{
		"height": 3,
		"width": 9,
		"yPos": 5,
		"widget": {
			"title": "Error budget burnrate on 12h sliding windows - Alert when > 3",
			"xyChart": {
				"chartOptions": {
					"mode": "COLOR"
				},
				"dataSets": [
					{
						"plotType": "LINE",
						"timeSeriesQuery": {
							"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n|filter metric.microservice_name == 'stream2bq'\n| filter metric.origin == '<origin>'\n| align delta(1m)\n| every 1m\n| group_by [metric.microservice_name], sliding(12h)\n| fraction_less_than_from <thresholdSeconds>\n| neg\n| add 1\n| div 0.01\n| cast_units \"1\""
						}
					}
				],
				"thresholds": [
					{
						"value": 3.0
					}
				],
				"timeshiftDuration": "0s",
				"yAxis": {
					"label": "y1Axis",
					"scale": "LINEAR"
				}
			}
		}
	},
	{
		"height": 3,
		"width": 9,
		"yPos": 8,
		"widget": {
			"title": "Error budget burnrate on 1h sliding windows - Alert when > 9",
			"xyChart": {
				"chartOptions": {
					"mode": "COLOR"
				},
				"dataSets": [
					{
						"plotType": "LINE",
						"timeSeriesQuery": {
							"timeSeriesQueryLanguage": "fetch cloud_function::logging.googleapis.com/user/ram_latency_e2e\n| filter (metric.microservice_name == 'stream2bq')\n| filter metric.origin == '<origin>'\n| align delta(28d)\n| every 28d\n| within 28d\n| group_by [metric.microservice_name]\n| graph_period 28d"
						}
					}
				],
				"thresholds": [
					{
						"value": 9.0
					}
				],
				"timeshiftDuration": "0s",
				"yAxis": {
					"label": "y1Axis",
					"scale": "LINEAR"
				}
			}
		}
	}
]
`
