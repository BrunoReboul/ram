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

package glo

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/BrunoReboul/ram/utilities/ffo"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"

	"github.com/BrunoReboul/ram/utilities/itst"
)

const testLogMetricPrefix = "ram_test_"

var projectsMetricsService *logging.ProjectsMetricsService

func TestIntegLogMetricDeployment_Deploy(t *testing.T) {
	var err error
	var creds *google.Credentials
	var core deploy.Core
	core.SolutionSettings.Hosting.ProjectID, creds = itst.GetIntegrationTestsProjectID()
	core.Ctx = context.Background()
	core.Services.LoggingService, err = logging.NewService(core.Ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalln(err)
	}
	projectsMetricsService = logging.NewProjectsMetricsService(core.Services.LoggingService)

	// Clean up gefore testing
	removeTestLogMetrics(core)

	testCases := []struct {
		name       string
		metricPath string
	}{
		{
			name:       "Step1_CreateLogMetric",
			metricPath: "testdata/test_metric_1.yaml",
		},
		{
			name: "Step2_UpdateLogMetric",
		},
		{
			name: "Step3_LogMetricIdUptodate",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logMetricDeployment := NewLogMetricDeployment()
			logMetricDeployment.Core = &core
			err := ffo.ReadUnmarshalYAML(tc.metricPath, &logMetricDeployment.Settings.Instance.GLO)
			if err != nil {
				log.Fatalln(err)
			}
			// logMetricDeployment.Settings.Instance.MON.Columns = tc.columns
			// logMetricDeployment.Artifacts.Widgets = []*monitoring.Widget{}
			// for _, microserviceName := range tc.microserviceNameList {
			// 	for _, widgetType := range tc.widgetTypeList {
			// 		widget, err := GetGCFWidget(microserviceName, widgetType)
			// 		if err != nil {
			// 			t.Fatalf("mon.GetGCFWidget %v", err)
			// 		}
			// 		logMetricDeployment.Artifacts.Widgets = append(logMetricDeployment.Artifacts.Widgets, &widget)
			// 	}
			// }
			// var buffer bytes.Buffer
			// log.SetOutput(&buffer)
			// defer func() {
			// 	log.SetOutput(os.Stderr)
			// }()
			// err := logMetricDeployment.Deploy()
			// msgString := buffer.String()
			// if err != nil {
			// 	t.Fatalf("logMetricDeployment.Deploy %v", err)
			// }
			// if strings.Contains(msgString, tc.wantMsgContains) {
			// 	t.Logf("OK got msg '%s'", msgString)
			// } else {
			// 	t.Errorf("want msg to contains '%s' and got \n'%s'", tc.wantMsgContains, msgString)
			// }
		})
	}
	// Clean up after testing
	removeTestLogMetrics(core)
}

func removeTestLogMetrics(core deploy.Core) {
	parent := fmt.Sprintf("projects/%s", core.SolutionSettings.Hosting.ProjectID)
	err := projectsMetricsService.List(parent).Pages(core.Ctx, browseRemoveTestLogMetrics)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func browseRemoveTestLogMetrics(response *logging.ListLogMetricsResponse) error {
	for _, logMetric := range response.Metrics {
		if strings.Contains(logMetric.Name, testLogMetricPrefix) {
			_, err := projectsMetricsService.Delete(logMetric.Name).Context(context.Background()).Do()
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("Deleted test log metric %s", logMetric.Name)
		}
	}
	return nil
}
