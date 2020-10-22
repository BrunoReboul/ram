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
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/monitoring/v1"
	"google.golang.org/api/option"

	"github.com/BrunoReboul/ram/utilities/itst"
)

const testDashboardDisplayName = "test dashboard"

var dashboardService *monitoring.ProjectsDashboardsService

func TestIntegDashboardDeployment_Deploy(t *testing.T) {
	var err error
	var creds *google.Credentials
	var core deploy.Core
	core.SolutionSettings.Hosting.Stackdriver.ProjectID, creds = itst.GetIntegrationTestsProjectID()
	core.Ctx = context.Background()
	core.Services.MonitoringService, err = monitoring.NewService(core.Ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	dashboardService = monitoring.NewProjectsDashboardsService(core.Services.MonitoringService)
	// Clean up gefore testing
	removeTestDashboards(core)

	testCases := []struct {
		name                 string
		microserviceNameList []string
		widgetTypeList       []string
		columns              int64
		wantMsgContains      string
	}{
		{
			name:                 "Step1_CreateDashboard",
			microserviceNameList: []string{"svc1", "svc2", "svc3"},
			widgetTypeList:       []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"},
			columns:              4,
			wantMsgContains:      "mon dashboard created",
		},
		{
			name:                 "Step2_UpdateDashboard",
			microserviceNameList: []string{"svc1", "svc2"},
			widgetTypeList:       []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"},
			columns:              4,
			wantMsgContains:      "mon dashboard updated",
		},
		{
			name:                 "Step3_DashboardIdUptodate",
			microserviceNameList: []string{"svc1", "svc2"},
			widgetTypeList:       []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"},
			columns:              4,
			wantMsgContains:      "mon dashboard is up-to-date",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dashboardDeployment := NewDashboardDeployment()
			dashboardDeployment.Core = &core
			dashboardDeployment.Settings.Instance.MON.DisplayName = testDashboardDisplayName
			dashboardDeployment.Settings.Instance.MON.Columns = tc.columns
			dashboardDeployment.Artifacts.Widgets = []*monitoring.Widget{}
			for _, microserviceName := range tc.microserviceNameList {
				for _, widgetType := range tc.widgetTypeList {
					widget, err := GetGCFWidget(microserviceName, widgetType)
					if err != nil {
						t.Fatalf("mon.GetGCFWidget %v", err)
					}
					dashboardDeployment.Artifacts.Widgets = append(dashboardDeployment.Artifacts.Widgets, &widget)
				}
			}
			var buffer bytes.Buffer
			log.SetOutput(&buffer)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			err := dashboardDeployment.Deploy()
			msgString := buffer.String()
			if err != nil {
				t.Fatalf("dashboardDeployment.Deploy %v", err)
			}
			if strings.Contains(msgString, tc.wantMsgContains) {
				t.Logf("OK got msg '%s'", msgString)
			} else {
				t.Errorf("want msg to contains '%s' and got \n'%s'", tc.wantMsgContains, msgString)
			}
		})
	}
	// Clean up after testing
	removeTestDashboards(core)
}

func removeTestDashboards(core deploy.Core) {
	parent := fmt.Sprintf("projects/%s", core.SolutionSettings.Hosting.Stackdriver.ProjectID)
	err := dashboardService.List(parent).Pages(core.Ctx, browseRemoveTestDashboards)
	if err != nil {
		log.Fatalln(err)
	}

	return
}

func browseRemoveTestDashboards(response *monitoring.ListDashboardsResponse) error {
	for _, dashboard := range response.Dashboards {
		if dashboard.DisplayName == testDashboardDisplayName {
			_, err := dashboardService.Delete(dashboard.Name).Context(context.Background()).Do()
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("Deleted test report %s", dashboard.Name)
		}
	}
	return nil
}
