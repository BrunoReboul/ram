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
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/logging/v2"
)

// Deploy log based metric
func (logMetricDeployment LogMetricDeployment) Deploy() (err error) {
	log.Printf("%s glo log metric", logMetricDeployment.Core.InstanceName)
	projectMetricsService := logging.NewProjectsMetricsService(logMetricDeployment.Core.Services.LoggingService)

	metricFound := true
	// GET
	// ffo.MarshalYAMLWrite("./metric.yaml", logMetricDeployment.Artifacts.LogMetric)
	retrievedLogMetric, err := projectMetricsService.Get(logMetricDeployment.Artifacts.LogMetric.Name).Context(logMetricDeployment.Core.Ctx).Do()

	if err != nil {
		fmt.Println(err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			metricFound = false
		} else {
			return fmt.Errorf("projectMetricsService.Get %v", err)
		}
	}

	if !metricFound {
		if logMetricDeployment.Core.Commands.Check {
			return fmt.Errorf("%s glo log based metric NOT found for this instance", logMetricDeployment.Core.InstanceName)
		}
		log.Printf("%s glo create metric start", logMetricDeployment.Core.InstanceName)
		parent := fmt.Sprintf("projects/%s", logMetricDeployment.Core.SolutionSettings.Hosting.ProjectID)
		createdLogMetric, err := projectMetricsService.Create(parent, &logMetricDeployment.Artifacts.LogMetric).Context(logMetricDeployment.Core.Ctx).Do()
		if err != nil {
			return fmt.Errorf("projectMetricsService.Create %v", err)
		}
		log.Printf("%s glo metric created %s", logMetricDeployment.Core.InstanceName, createdLogMetric.Name)
	} else {
		log.Printf("%s glo found log metric %s",
			logMetricDeployment.Core.InstanceName,
			retrievedLogMetric.Name)
		err = checkLogMetric(&logMetricDeployment.Artifacts.LogMetric, retrievedLogMetric)
		if err != nil {
			if logMetricDeployment.Core.Commands.Check {
				return err
			}
			log.Printf("%s glo metric meed to be updated", logMetricDeployment.Core.InstanceName)
			updatedLogMetric, err := projectMetricsService.Update(logMetricDeployment.Artifacts.LogMetric.Name, &logMetricDeployment.Artifacts.LogMetric).Context(logMetricDeployment.Core.Ctx).Do()
			if err != nil {
				return fmt.Errorf("projectMetricsService.Update %v", err)
			}
			log.Printf("%s glo metric updated %s", logMetricDeployment.Core.InstanceName, updatedLogMetric.Name)
		}
	}
	return nil
}
