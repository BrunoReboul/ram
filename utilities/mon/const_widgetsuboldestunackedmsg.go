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

const widgetSubOldestUnackedMsg = `
{
	"title": "mservice_name pubsub oldest unacked msg",
	"xyChart": {
	  "chartOptions": {
		"mode": "COLOR"
	  },
	  "dataSets": [
		{
		  "minAlignmentPeriod": "60s",
		  "plotType": "LINE",
		  "timeSeriesQuery": {
			"timeSeriesFilter": {
			  "aggregation": {
				"perSeriesAligner": "ALIGN_MEAN"
			  },
			  "filter": "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\" resource.type=\"pubsub_subscription\" resource.label.\"subscription_id\"=monitoring.regex.full_match(\".*mservice_name.*\")",
			  "secondaryAggregation": {}
			}
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
`
