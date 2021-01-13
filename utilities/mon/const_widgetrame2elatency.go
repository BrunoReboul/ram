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

const widgetRAMe2eLatency = `
{
  "title": "microservice_name end to end latency",
  "xyChart": {
    "chartOptions": {
      "mode": "COLOR"
    },
    "dataSets": [
      {
        "minAlignmentPeriod": "60s",
        "plotType": "HEATMAP",
        "timeSeriesQuery": {
          "timeSeriesFilter": {
            "aggregation": {
              "crossSeriesReducer": "REDUCE_SUM",
              "perSeriesAligner": "ALIGN_DELTA"
            },
            "filter": "metric.type=\"logging.googleapis.com/user/ram_latency_e2e\" resource.type=\"cloud_function\" metric.label.\"microservice_name\"=\"microservice_name\"",
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
