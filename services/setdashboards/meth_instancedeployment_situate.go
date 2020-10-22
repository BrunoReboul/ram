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
	"github.com/BrunoReboul/ram/utilities/mon"
	"google.golang.org/api/monitoring/v1"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	instanceDeployment.Artifacts.Widgets = []*monitoring.Widget{}
	for _, microserviceName := range instanceDeployment.Settings.Instance.MON.MicroServiceNameList {
		for _, widgetType := range instanceDeployment.Settings.Instance.MON.WidgetTypeList {
			widget, err := mon.GetGCFWidget(microserviceName, widgetType)
			if err != nil {
				return err
			}
			instanceDeployment.Artifacts.Widgets = append(instanceDeployment.Artifacts.Widgets, &widget)
		}
	}
	return nil
}
