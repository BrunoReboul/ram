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

var dashboardService *monitoring.ProjectsDashboardsService

func TestIntegDeployment_DeployMonitoringDashboards(t *testing.T) {
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
		name         string
		wantMessages []string
	}{
		{
			name:         "Step1CreateDashboards",
			wantMessages: []string{"mon dashboard created '" + coreDashboardName + "'", "mon dashboard created '" + groupsDashboardName + "'"},
		},
		{
			name:         "Step2DashboardsAreUptodate",
			wantMessages: []string{"mon dashboard is up-to-date '" + coreDashboardName + "'", "mon dashboard is up-to-date '" + groupsDashboardName + "'"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var deployment Deployment
			deployment.Core = core
			var buffer bytes.Buffer
			log.SetOutput(&buffer)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			err := deployment.deployMonitoringDashboards()
			msgString := buffer.String()
			if err != nil {
				t.Fatalf("deployment.deployMonitoringDashboards %v", err)
			}
			for _, wantMsgContains := range tc.wantMessages {
				if strings.Contains(msgString, wantMsgContains) {
					t.Logf("OK got msg '%s'", wantMsgContains)
				} else {
					t.Errorf("want msg to contains '%s' and got \n'%s'", wantMsgContains, msgString)
				}
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
		if dashboard.DisplayName == coreDashboardName || dashboard.DisplayName == groupsDashboardName {
			_, err := dashboardService.Delete(dashboard.Name).Context(context.Background()).Do()
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("Deleted test report '%s' %s", dashboard.DisplayName, dashboard.Name)
		}
	}
	return nil
}
