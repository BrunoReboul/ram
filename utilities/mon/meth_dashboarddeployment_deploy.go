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
	"fmt"
	"log"
	"reflect"
	"strings"

	"google.golang.org/api/monitoring/v1"
)

var dashboardID, dashboardDisplayName string

// Deploy dashboard
func (dashboardDeployment DashboardDeployment) Deploy() (err error) {
	dashboardService := monitoring.NewProjectsDashboardsService(dashboardDeployment.Core.Services.MonitoringService)
	parent := fmt.Sprintf("projects/%s", dashboardDeployment.Core.SolutionSettings.Hosting.Stackdriver.ProjectID)
	dashboardDisplayName = dashboardDeployment.Settings.DisplayName
	dashboardID = ""
	err = dashboardService.List(parent).Pages(dashboardDeployment.Core.Ctx, browseDashboards)
	if err != nil {
		if err.Error() != "found_dashboard" {
			return fmt.Errorf("dashboardService.List %v", err)
		}
	}
	var gridLayout monitoring.GridLayout
	gridLayout.Columns = dashboardDeployment.Settings.Columns
	gridLayout.Widgets = dashboardDeployment.Settings.Widgets
	var dashboard monitoring.Dashboard
	dashboard.DisplayName = dashboardDeployment.Settings.DisplayName
	dashboard.GridLayout = &gridLayout
	if dashboardID == "" {
		// Create dashboard
		retreivedDashboard, err := dashboardService.Create(parent, &dashboard).Context(dashboardDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
		log.Printf("mon dashboard created '%s' %s", retreivedDashboard.DisplayName, retreivedDashboard.Name)
	} else {
		// Patch dashboard
		dashboardName := fmt.Sprintf("projects/%s/dashboards/%s",
			dashboardDeployment.Core.SolutionSettings.Hosting.Stackdriver.ProjectID,
			dashboardID)
		retreivedDashboard, err := dashboardService.Get(dashboardName).Context(dashboardDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(dashboardDeployment.Settings.Widgets, retreivedDashboard.GridLayout.Widgets) {
			dashboard.Etag = retreivedDashboard.Etag
			retreivedDashboard, err := dashboardService.Patch(dashboardName, &dashboard).Context(dashboardDeployment.Core.Ctx).Do()
			if err != nil {
				return err
			}
			log.Printf("mon dashboard updated '%s' %s", retreivedDashboard.DisplayName, retreivedDashboard.Name)
		} else {
			log.Printf("mon dashboard is up-to-date '%s'", dashboardDeployment.Settings.DisplayName)
		}
	}

	return nil
}

func browseDashboards(response *monitoring.ListDashboardsResponse) error {
	for _, dashboard := range response.Dashboards {
		if dashboard.DisplayName == dashboardDisplayName {
			parts := strings.Split(dashboard.Name, "/")
			dashboardID = parts[len(parts)-1]
			return fmt.Errorf("found_dashboard")
		}
	}
	return nil
}
