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
	"log"
	"testing"

	"google.golang.org/api/logging/v2"
	"gopkg.in/yaml.v2"
)

func TestUnitCheckLogMetrics(t *testing.T) {
	type testcases []struct {
		Name               string
		WantErr            bool              `yaml:"wantErr"`
		LogMetric          logging.LogMetric `yaml:"logMetric"`
		RetreivedLogMetric logging.LogMetric `yaml:"retreivedLogMetric"`
	}
	var testCases testcases

	yamlBytes := []byte(`---
- name: identical full
  wantErr: false
  logMetric:
    bucketoptions:
      explicitbuckets: null
      exponentialbuckets:
        growthfactor: 1.4142135623731
        numfinitebuckets: 64
        scale: 0.01
        forcesendfields: []
        nullfields: []
      linearbuckets: null
    description: RAM latency by component
    filter: resource.type="cloud_function" severity=NOTICE jsonPayload.message=~"^finish"
    labelextractors:
      environment: EXTRACT(jsonPayload.environment)
      instance_name: EXTRACT(jsonPayload.instance_name)
      microservice_name: EXTRACT(jsonPayload.microservice_name)
    metricdescriptor:
      description: RAM latency by component
      displayname: ""
      labels:
      - description: dev, prd...
        key: environment
      - description: instance name
        key: instance_name
      - description: microservice name
        key: microservice_name
      launchstage: ""
      metrickind: DELTA
      name: projects/brunore-ram-dev-003/metricDescriptors/logging.googleapis.com/user/ram_latency
      type: logging.googleapis.com/user/ram_latency
      unit: s
      valuetype: DISTRIBUTION
    name: projects/brunore-ram-dev-003/metrics/ram_latency
    valueextractor: EXTRACT(jsonPayload.latency_seconds)
  retreivedLogMetric:
    bucketoptions:
      explicitbuckets: null
      exponentialbuckets:
        growthfactor: 1.4142135623731
        numfinitebuckets: 64
        scale: 0.01
        forcesendfields: []
        nullfields: []
      linearbuckets: null
    description: RAM latency by component
    filter: resource.type="cloud_function" severity=NOTICE jsonPayload.message=~"^finish"
    labelextractors:
      environment: EXTRACT(jsonPayload.environment)
      instance_name: EXTRACT(jsonPayload.instance_name)
      microservice_name: EXTRACT(jsonPayload.microservice_name)
    metricdescriptor:
      description: RAM latency by component
      displayname: ""
      labels:
      - description: dev, prd...
        key: environment
      - description: instance name
        key: instance_name
      - description: microservice name
        key: microservice_name
      launchstage: ""
      metrickind: DELTA
      name: projects/brunore-ram-dev-003/metricDescriptors/logging.googleapis.com/user/ram_latency
      type: logging.googleapis.com/user/ram_latency
      unit: s
      valuetype: DISTRIBUTION
    name: projects/brunore-ram-dev-003/metrics/ram_latency
    valueextractor: EXTRACT(jsonPayload.latency_seconds)
- name: description
  wantErr: true
  logMetric:
    description: A
  retreivedLogMetric:
    description: B
- name: filter
  wantErr: true
  logMetric:
    filter: A
  retreivedLogMetric:
    filter: B
- name: valueExtractor
  wantErr: true
  logMetric:
    valueextractor: A
  retreivedLogMetric:
    valueextractor: B
- name: LabelExtractors
  wantErr: true
  logMetric:
    labelextractors:
      environment: A
  retreivedLogMetric:
- name: LabelExtractor
  wantErr: true
  logMetric:
    labelextractors:
      environment: A
  retreivedLogMetric:
    labelextractors:
      environment: B
- name: ExplicitBuckets
  wantErr: true
  logMetric:
    bucketoptions:
      explicitbuckets:
        bounds:
          - 1
          - 2
  retreivedLogMetric:
    bucketoptions:
      exponentialbuckets:
        growthfactor: 2
- name: bounds
  wantErr: true
  logMetric:
    bucketoptions:
      explicitbuckets:
        bounds:
          - 1
          - 2
  retreivedLogMetric:
    bucketoptions:
      explicitbuckets:
        bounds:
          - 1
- name: ExponentialBuckets
  wantErr: true
  logMetric:
    bucketoptions:
      exponentialbuckets:
        growthfactor: 2
  retreivedLogMetric:
    bucketoptions:
      explicitbuckets:
        bounds:
          - 1
- name: growthfactor
  wantErr: true
  logMetric:
    bucketoptions:
      exponentialbuckets:
        growthfactor: 2
  retreivedLogMetric:
    bucketoptions:
      exponentialbuckets:
        growthfactor: 10
- name: expo numfinitebuckets
  wantErr: true
  logMetric:
    bucketoptions:
      exponentialbuckets:
         numfinitebuckets: 2
  retreivedLogMetric:
    bucketoptions:
      exponentialbuckets:
         numfinitebuckets: 10
- name: scale
  wantErr: true
  logMetric:
    bucketoptions:
      exponentialbuckets:
         scale: 2
  retreivedLogMetric:
    bucketoptions:
      exponentialbuckets:
         scale: 10
- name: linearbuckets
  wantErr: true
  logMetric:
    bucketoptions:
      linearbuckets:
         numfinitebuckets: 2
  retreivedLogMetric:
    bucketoptions:
      explicitbuckets:
        bounds:
          - 1
- name: linear numfinitebuckets
  wantErr: true
  logMetric:
    bucketoptions:
      linearbuckets:
         numfinitebuckets: 2
  retreivedLogMetric:
    bucketoptions:
      linearbuckets:
         numfinitebuckets: 10
- name: Offset
  wantErr: true
  logMetric:
    bucketoptions:
      linearbuckets:
         offset: 2
  retreivedLogMetric:
    bucketoptions:
      linearbuckets:
         offset: 10
- name: Width
  wantErr: true
  logMetric:
    bucketoptions:
      linearbuckets:
         width: 2
  retreivedLogMetric:
    bucketoptions:
      linearbuckets:
         width: 10
- name: MetricDescriptor
  wantErr: true
  logMetric:
    metricdescriptor:
      description: A
  retreivedLogMetric:
- name: MetricDescriptor type
  wantErr: true
  logMetric:
    metricdescriptor:
      type: A
  retreivedLogMetric:
    metricdescriptor:
      type: B
- name: MetricDescriptor description
  wantErr: true
  logMetric:
    metricdescriptor:
      description: A
  retreivedLogMetric:
    metricdescriptor:
      description: B
- name: MetricDescriptor launchStage
  wantErr: true
  logMetric:
    metricdescriptor:
      launchstage: A
  retreivedLogMetric:
    metricdescriptor:
      launchstage: B
- name: MetricDescriptor metricKind
  wantErr: true
  logMetric:
    metricdescriptor:
      metrickind: A
  retreivedLogMetric:
    metricdescriptor:
      metrickind: B
- name: MetricDescriptor unit
  wantErr: true
  logMetric:
    metricdescriptor:
      unit: A
  retreivedLogMetric:
    metricdescriptor:
      unit: B
- name: MetricDescriptor valueType
  wantErr: true
  logMetric:
    metricdescriptor:
      valuetype: A
  retreivedLogMetric:
    metricdescriptor:
      valuetype: B
- name: MetricDescriptor labels
  wantErr: true
  logMetric:
    metricdescriptor:
      labels:
      - description: A
  retreivedLogMetric:
    metricdescriptor:
- name: MetricDescriptor labels description
  wantErr: true
  logMetric:
    metricdescriptor:
      labels:
      - description: A
  retreivedLogMetric:
    metricdescriptor:
      labels:
      - description: B`)

	err := yaml.Unmarshal(yamlBytes, &testCases)
	if err != nil {
		log.Fatalf("Unable to unmarshal yaml test data %v", err)
	}

	for _, tt := range testCases {
		tt := tt // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			err := checkLogMetric(&tt.LogMetric, &tt.RetreivedLogMetric)
			if err != nil {
				t.Log(err.Error())
			}
			if (err != nil) != tt.WantErr {
				t.Errorf("error = %v, WantErr %v", err, tt.WantErr)
				return
			}
		})
	}
}
