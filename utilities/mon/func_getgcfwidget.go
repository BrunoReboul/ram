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

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/api/monitoring/v1"
)

//GetGCFWidget assemble a monitoring widget for given widget type and microservice name
func GetGCFWidget(microserviceName string, widgetType string) (widget monitoring.Widget, err error) {
	var widgetTypeJSON string
	switch widgetType {
	case "widgetGCFActiveInstances":
		widgetTypeJSON = widgetGCFActiveInstances
	case "widgetGCFExecutionCount":
		widgetTypeJSON = widgetGCFExecutionCount
	case "widgetGCFExecutionTime":
		widgetTypeJSON = widgetGCFExecutionTime
	case "widgetGCFMemoryUsage":
		widgetTypeJSON = widgetGCFMemoryUsage
	case "widgetRAMe2eLatency":
		widgetTypeJSON = widgetRAMe2eLatency
	case "widgetRAMLatency":
		widgetTypeJSON = widgetRAMLatency
	case "widgetRAMTriggerAge":
		widgetTypeJSON = widgetRAMTriggerAge
	default:
		return widget, fmt.Errorf("Unsupported widgetType")
	}
	if microserviceName == "" {
		return widget, fmt.Errorf("microserviceName can NOT be a zero value")
	}
	widgetTypeJSON = strings.Replace(widgetTypeJSON, "mservice_name", microserviceName, -1)
	err = json.Unmarshal([]byte(widgetTypeJSON), &widget)
	if err != nil {
		return widget, fmt.Errorf("json.Unmarshal %s %s %v", microserviceName, widgetType, err)
	}
	return widget, nil
}
