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

// LogMetricParameters structure
type LogMetricParameters struct {
	MetricID       string `yaml:"metric_id"`
	Description    string
	Filter         string
	ValueExtractor string `yaml:"valueExtractor,omitempty"`
	BucketOptions  struct {
		ExplicitBuckets struct {
			Bounds []float64 `yaml:"bounds,omitempty"`
		} `yaml:"explicitBuckets,omitempty"`
		ExponentialBuckets struct {
			GrowthFactor     float64 `yaml:"growthFactor,omitempty"`
			NumFiniteBuckets int64   `yaml:"numFiniteBuckets,omitempty"`
			Scale            float64 `yaml:"scale,omitempty"`
		} `yaml:"exponentialBuckets,omitempty"`
		LinearBuckets struct {
			NumFiniteBuckets int64   `yaml:"numFiniteBuckets,omitempty"`
			Offset           float64 `yaml:"offset,omitempty"`
			Width            float64 `yaml:"width,omitempty"`
		} `yaml:"linearBuckets,omitempty"`
	} `yaml:"bucketOptions,omitempty"`
	Labels []struct {
		Name        string `yaml:"name,omitempty"`
		Extractor   string `yaml:"extractor,omitempty"`
		Description string `yaml:"description,omitempty"`
		ValueType   string `yaml:"valueType,omitempty"`
	} `yaml:"labels,omitempty"`
	MetricDescriptor struct {
		LaunchStage string `yaml:"launchStage,omitempty"`
		MetricKind  string `yaml:"metricKind,omitempty"`
		Unit        string `yaml:"unit,omitempty"`
		ValueType   string `yaml:"valueType,omitempty"`
	} `yaml:"metricDescriptor,omitempty"`
}
