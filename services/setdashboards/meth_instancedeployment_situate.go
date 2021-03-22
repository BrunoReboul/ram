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

package setdashboards

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/BrunoReboul/ram/utilities/mon"
	"google.golang.org/api/monitoring/v1"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	instanceDeployment.Artifacts.Widgets = []*monitoring.Widget{}
	instanceDeployment.Artifacts.Tiles = []*monitoring.Tile{}
	if instanceDeployment.Settings.Instance.MON.GridLayout.Columns != 0 {
		for _, microserviceName := range instanceDeployment.Settings.Instance.MON.GridLayout.MicroServiceNameList {
			for _, widgetType := range instanceDeployment.Settings.Instance.MON.GridLayout.WidgetTypeList {
				widget, err := mon.GetGCFWidget(microserviceName, widgetType)
				if err != nil {
					return err
				}
				instanceDeployment.Artifacts.Widgets = append(instanceDeployment.Artifacts.Widgets, &widget)
			}
		}
	}
	if instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.SLO != 0 {
		grouwthFactor := math.Sqrt(2)
		scale := 0.01
		thresholdSeconds := scale * math.Pow(grouwthFactor, float64(instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.CutOffBucketNumber))
		var thresholdText string
		if thresholdSeconds < 60 {
			thresholdText = fmt.Sprintf("%g seconds", math.Round(thresholdSeconds))
		} else {
			if thresholdSeconds < 60*60 {
				thresholdText = fmt.Sprintf("%g minutes", math.Round(thresholdSeconds/60))
			} else {
				if thresholdSeconds < 60*60*60 {
					thresholdText = fmt.Sprintf("%g hours", math.Round(thresholdSeconds/60/60))
				}
			}
		}
		slo := instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.SLO
		sloText := fmt.Sprintf("%g%%", slo*100)
		dashboardJSON := mon.SLOFreshnessTiles
		dashboardJSON = strings.Replace(dashboardJSON, "<origin>", instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.Origin, -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<scope>", instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.Scope, -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<flow>", instanceDeployment.Settings.Instance.MON.SLOFreshnessLayout.Flow, -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<slo>", fmt.Sprintf("%g", slo), -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<lowerBound>", fmt.Sprintf("%g", math.Floor(slo*10)/10), -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<thresholdSeconds>", fmt.Sprintf("%v", thresholdSeconds), -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<thresholdText>", thresholdText, -1)
		dashboardJSON = strings.Replace(dashboardJSON, "<sloText>", sloText, -1)
		err = json.Unmarshal([]byte(dashboardJSON), &instanceDeployment.Artifacts.Tiles)
		if err != nil {
			return fmt.Errorf("json.Unmarshal SLOFreshnessTiles %v", err)
		}
	}
	return nil
}
