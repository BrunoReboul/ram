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

package ramcli

import (
	"github.com/BrunoReboul/ram/utilities/mon"
	"google.golang.org/api/monitoring/v1"
)

func (deployment *Deployment) deployMonitoringDashboards() (err error) {
	dashboardDeployment := mon.NewDashboardDeployment()

	// core report
	dashboardDeployment.Core = &deployment.Core
	dashboardDeployment.Settings.DisplayName = "RAM core microservices"
	dashboardDeployment.Settings.Columns = 4
	dashboardDeployment.Settings.Widgets = []*monitoring.Widget{}
	for _, microserviceName := range []string{"dumpinventory", "splitdump", "monitor", "stream2bq", "publish2fs", "upload2gcs"} {
		for _, widgetType := range []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"} {
			widget, err := mon.GetGCFWidget(microserviceName, widgetType)
			if err != nil {
				return err
			}
			dashboardDeployment.Settings.Widgets = append(dashboardDeployment.Settings.Widgets, &widget)
		}
	}
	if err := dashboardDeployment.Deploy(); err != nil {
		return err
	}

	// groups report
	dashboardDeployment.Core = &deployment.Core
	dashboardDeployment.Settings.DisplayName = "RAM groups microservices"
	dashboardDeployment.Settings.Columns = 4
	dashboardDeployment.Settings.Widgets = []*monitoring.Widget{}
	for _, microserviceName := range []string{"convertlog2feed", "listgroups", "getgroupsettings", "listgroupmembers"} {
		// for _, widgetType := range []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"} {
		for _, widgetType := range []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"} {
			widget, err := mon.GetGCFWidget(microserviceName, widgetType)
			if err != nil {
				return err
			}
			dashboardDeployment.Settings.Widgets = append(dashboardDeployment.Settings.Widgets, &widget)
		}
	}
	if err := dashboardDeployment.Deploy(); err != nil {
		return err
	}

	return nil
}
