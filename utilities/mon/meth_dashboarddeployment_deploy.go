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

	// "reflect"
	"strings"

	"google.golang.org/api/monitoring/v1"
)

var dashboardID, dashboardDisplayName string

// Deploy dashboard
func (dashboardDeployment DashboardDeployment) Deploy() (err error) {
	dashboardService := monitoring.NewProjectsDashboardsService(dashboardDeployment.Core.Services.MonitoringService)
	parent := fmt.Sprintf("projects/%s", dashboardDeployment.Core.SolutionSettings.Hosting.Stackdriver.ProjectID)
	dashboardDisplayName = dashboardDeployment.Settings.Instance.MON.DisplayName
	dashboardID = ""
	err = dashboardService.List(parent).Pages(dashboardDeployment.Core.Ctx, browseDashboards)
	if err != nil {
		if err.Error() != "found_dashboard" {
			return fmt.Errorf("dashboardService.List %v", err)
		}
	}
	// var gridLayout monitoring.GridLayout
	// var mosaicLayout monitoring.MosaicLayout
	var dashboard monitoring.Dashboard
	var dashboardName string
	var retreivedDashboard *monitoring.Dashboard
	needToUpdate := false
	dashboard.DisplayName = dashboardDeployment.Settings.Instance.MON.DisplayName

	if dashboardID != "" {
		dashboardName = fmt.Sprintf("projects/%s/dashboards/%s",
			dashboardDeployment.Core.SolutionSettings.Hosting.Stackdriver.ProjectID,
			dashboardID)
		retreivedDashboard, err = dashboardService.Get(dashboardName).Context(dashboardDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
	} else {
		if dashboardDeployment.Core.Commands.Check {
			return fmt.Errorf("%s mon dashboard NOT found for this instance", dashboardDeployment.Core.InstanceName)
		}
	}

	// if dashboardDeployment.Settings.Instance.MON.GridLayout.Columns != 0 {
	// 	// Gridlayout report
	// 	gridLayout.Columns = dashboardDeployment.Settings.Instance.MON.GridLayout.Columns
	// 	gridLayout.Widgets = dashboardDeployment.Artifacts.Widgets
	// 	dashboard.GridLayout = &gridLayout
	// 	needToUpdateWidgets := false
	// 	needToUpdateColumns := false
	// 	if dashboardID != "" {
	// 		if !reflect.DeepEqual(dashboardDeployment.Artifacts.Widgets, retreivedDashboard.GridLayout.Widgets) {
	// 			needToUpdateWidgets = true
	// 			needToUpdate = true
	// 		}
	// 		if dashboardDeployment.Settings.Instance.MON.GridLayout.Columns != retreivedDashboard.GridLayout.Columns {
	// 			needToUpdateColumns = true
	// 			needToUpdate = true
	// 		}
	// 	}

	// 	if dashboardDeployment.Core.Commands.Check {
	// 		var s string
	// 		if needToUpdateWidgets {
	// 			s = fmt.Sprintf("%shave a different array of widgets\n", s)
	// 		}
	// 		if needToUpdateColumns {
	// 			s = fmt.Sprintf("%scolumns\nwant %d\nhave %d\n", s,
	// 				dashboardDeployment.Settings.Instance.MON.GridLayout.Columns,
	// 				retreivedDashboard.GridLayout.Columns)
	// 		}
	// 		if len(s) > 0 {
	// 			return fmt.Errorf("%s mon invalid dashboard configuration:\n%s", dashboardDeployment.Core.InstanceName, s)
	// 		}
	// 		return nil
	// 	}
	// } else {
	// 	if dashboardDeployment.Settings.Instance.MON.SLOFreshnessLayout.SLO != 0 {
	// 		SLO freshness mosaic report
	// 		mosaicLayout.Columns = dashboardDeployment.Settings.Instance.MON.SLOFreshnessLayout.Columns
	// 		mosaicLayout.Tiles = dashboardDeployment.Artifacts.Tiles
	// 		dashboard.MosaicLayout = &mosaicLayout
	// 		needToUpdateTiles := false
	// 		needToUpdateColumns := false
	// 		if dashboardID != "" {
	// 			if !reflect.DeepEqual(dashboardDeployment.Artifacts.Tiles, retreivedDashboard.MosaicLayout.Tiles) {
	// 				needToUpdateTiles = true
	// 				needToUpdate = true
	// 			}
	// 			if dashboardDeployment.Settings.Instance.MON.SLOFreshnessLayout.Columns != retreivedDashboard.MosaicLayout.Columns {
	// 				needToUpdateColumns = true
	// 				needToUpdate = true
	// 			}
	// 		}
	// 		if dashboardDeployment.Core.Commands.Check {
	// 			var s string
	// 			if needToUpdateTiles {
	// 				s = fmt.Sprintf("%shave a different array of tiles\n", s)
	// 			}
	// 			if needToUpdateColumns {
	// 				s = fmt.Sprintf("%scolumns\nwant %d\nhave %d\n", s,
	// 					dashboardDeployment.Settings.Instance.MON.SLOFreshnessLayout.Columns,
	// 					retreivedDashboard.MosaicLayout.Columns)
	// 			}
	// 			if len(s) > 0 {
	// 				return fmt.Errorf("%s mon invalid dashboard configuration:\n%s", dashboardDeployment.Core.InstanceName, s)
	// 			}
	// 			return nil
	// 		// }

	// 	} else {
	// 		return fmt.Errorf("unmanged dashboard type")
	// 	}
	// }

	if dashboardID == "" {
		// Create dashboard
		retreivedDashboard, err = dashboardService.Create(parent, &dashboard).Context(dashboardDeployment.Core.Ctx).Do()
		if err != nil {
			return err
		}
		log.Printf("mon dashboard created '%s' %s", retreivedDashboard.DisplayName, retreivedDashboard.Name)
	} else {
		// Patch dashboard
		if needToUpdate {
			dashboard.Etag = retreivedDashboard.Etag
			retreivedDashboard, err = dashboardService.Patch(dashboardName, &dashboard).Context(dashboardDeployment.Core.Ctx).Do()
			if err != nil {
				return err
			}
			log.Printf("mon dashboard updated '%s' %s", retreivedDashboard.DisplayName, retreivedDashboard.Name)
		} else {
			log.Printf("mon dashboard is up-to-date '%s'", dashboardDeployment.Settings.Instance.MON.DisplayName)
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
