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
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BrunoReboul/ram/utilities/glo"
	"gopkg.in/yaml.v2"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureSetLogMetricss
func (deployment *Deployment) configureSetLogMetrics() (err error) {
	serviceName := "setlogmetrics"
	serviceFolderPath := fmt.Sprintf("%s/%s/%s",
		deployment.Core.RepositoryPath,
		solution.MicroserviceParentFolderName,
		serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}

	log.Printf("configure %s", serviceName)
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}
	logMetricListYAML := []byte(`
- glo:
    metric_id: ram_latency
    description: RAM latency by component
    filter: resource.type="cloud_function" severity=NOTICE jsonPayload.message=~"^finish"
    valueExtractor: EXTRACT(jsonPayload.latency_seconds)
    bucketOptions:
      exponentialBuckets:
        growthFactor: 1.4142135623731
        numFiniteBuckets: 64
        scale: 0.01
    labels:
      - name: environment
        extractor: EXTRACT(jsonPayload.environment)
        description: dev, prd...
        valueType: string
      - name: instance_name
        extractor: EXTRACT(jsonPayload.instance_name)
        description: instance name
        valueType: string
      - name: microservice_name
        extractor: EXTRACT(jsonPayload.microservice_name)
        description: microservice name
        valueType: string
    metricDescriptor:
      metricKind: DELTA
      unit: s
      valueType: DISTRIBUTION
- glo:
    metric_id: ram_latency_e2e
    description: RAM latency end to end
    filter: resource.type="cloud_function" severity=NOTICE jsonPayload.message=~"^finish"
    valueExtractor: EXTRACT(jsonPayload.latency_e2e_seconds)
    bucketOptions:
      exponentialBuckets:
        growthFactor: 1.4142135623731
        numFiniteBuckets: 64
        scale: 0.01
    labels:
      - name: environment
        extractor: EXTRACT(jsonPayload.environment)
        description: dev, prd...
        valueType: string
      - name: instance_name
        extractor: EXTRACT(jsonPayload.instance_name)
        description: instance name
        valueType: string
      - name: microservice_name
        extractor: EXTRACT(jsonPayload.microservice_name)
        description: microservice name
        valueType: string
    metricDescriptor:
      metricKind: DELTA
      unit: s
      valueType: DISTRIBUTION
- glo:
    metric_id: ram_trigger_age
    description: RAM age of the triggering event
    filter: resource.type="cloud_function" severity=NOTICE jsonPayload.message="start"
    valueExtractor: EXTRACT(jsonPayload.triggering_pubsub_age_seconds)
    bucketOptions:
      exponentialBuckets:
        growthFactor: 1.4142135623731
        numFiniteBuckets: 64
        scale: 0.01
    labels:
      - name: environment
        extractor: EXTRACT(jsonPayload.environment)
        description: dev, prd...
        valueType: string
      - name: instance_name
        extractor: EXTRACT(jsonPayload.instance_name)
        description: instance name
        valueType: string
      - name: microservice_name
        extractor: EXTRACT(jsonPayload.microservice_name)
        description: microservice name
        valueType: string
    metricDescriptor:
      metricKind: DELTA
      unit: s
      valueType: DISTRIBUTION`)

	var logMetricList []struct {
		GLO glo.LogMetricParameters
	}
	err = yaml.Unmarshal(logMetricListYAML, &logMetricList)
	if err != nil {
		return err
	}

	for _, logMetric := range logMetricList {
		instanceFolderPath := fmt.Sprintf("%s/%s_%s",
			instancesFolderPath,
			serviceName,
			strings.ToLower(strings.Replace(logMetric.GLO.MetricID, " ", "_", -1)))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s",
			instanceFolderPath,
			solution.InstanceSettingsFileName),
			logMetric); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
