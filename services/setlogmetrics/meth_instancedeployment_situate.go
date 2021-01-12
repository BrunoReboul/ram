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

package setlogmetrics

import (
	"fmt"

	"google.golang.org/api/logging/v2"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	metricName := fmt.Sprintf("projects/%s/metrics/%s",
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		instanceDeployment.Settings.Instance.GLO.MetricID)

	instanceDeployment.Artifacts.LogMetric.Name = metricName
	instanceDeployment.Artifacts.LogMetric.Description = instanceDeployment.Settings.Instance.GLO.Description
	instanceDeployment.Artifacts.LogMetric.Filter = instanceDeployment.Settings.Instance.GLO.Filter
	instanceDeployment.Artifacts.LogMetric.ValueExtractor = instanceDeployment.Settings.Instance.GLO.ValueExtractor

	var explicitBuckets logging.Explicit
	var exponentialBuckets logging.Exponential
	var linearBuckets logging.Linear
	var bucketOptions logging.BucketOptions
	instanceDeployment.Artifacts.LogMetric.BucketOptions = &bucketOptions
	if len(instanceDeployment.Settings.Instance.GLO.BucketOptions.ExplicitBuckets.Bounds) > 0 {
		bucketOptions.ExplicitBuckets = &explicitBuckets
		explicitBuckets.Bounds = instanceDeployment.Settings.Instance.GLO.BucketOptions.ExplicitBuckets.Bounds
	} else {
		if instanceDeployment.Settings.Instance.GLO.BucketOptions.ExponentialBuckets.NumFiniteBuckets != 0 {
			bucketOptions.ExponentialBuckets = &exponentialBuckets
			exponentialBuckets.GrowthFactor = instanceDeployment.Settings.Instance.GLO.BucketOptions.ExponentialBuckets.GrowthFactor
			exponentialBuckets.NumFiniteBuckets = instanceDeployment.Settings.Instance.GLO.BucketOptions.ExponentialBuckets.NumFiniteBuckets
			exponentialBuckets.Scale = instanceDeployment.Settings.Instance.GLO.BucketOptions.ExponentialBuckets.Scale
		} else {
			if instanceDeployment.Settings.Instance.GLO.BucketOptions.LinearBuckets.NumFiniteBuckets != 0 {
				bucketOptions.LinearBuckets = &linearBuckets
				linearBuckets.NumFiniteBuckets = instanceDeployment.Settings.Instance.GLO.BucketOptions.LinearBuckets.NumFiniteBuckets
				linearBuckets.Offset = instanceDeployment.Settings.Instance.GLO.BucketOptions.LinearBuckets.Offset
				linearBuckets.Width = instanceDeployment.Settings.Instance.GLO.BucketOptions.LinearBuckets.Width
			}
		}
	}

	var metricDescriptor logging.MetricDescriptor
	instanceDeployment.Artifacts.LogMetric.MetricDescriptor = &metricDescriptor
	metricDescriptor.Type = fmt.Sprintf("logging.googleapis.com/user/%s", instanceDeployment.Settings.Instance.GLO.MetricID)
	metricDescriptor.Name = fmt.Sprintf("projects/%s/metricDescriptors/logging.googleapis.com/user/%s",
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		instanceDeployment.Settings.Instance.GLO.MetricID)
	metricDescriptor.Description = instanceDeployment.Settings.Instance.GLO.Description
	metricDescriptor.LaunchStage = instanceDeployment.Settings.Instance.GLO.MetricDescriptor.LaunchStage
	metricDescriptor.MetricKind = instanceDeployment.Settings.Instance.GLO.MetricDescriptor.MetricKind
	metricDescriptor.Unit = instanceDeployment.Settings.Instance.GLO.MetricDescriptor.Unit
	metricDescriptor.ValueType = instanceDeployment.Settings.Instance.GLO.MetricDescriptor.ValueType

	instanceDeployment.Artifacts.LogMetric.LabelExtractors = make(map[string]string)
	for _, label := range instanceDeployment.Settings.Instance.GLO.Labels {
		instanceDeployment.Artifacts.LogMetric.LabelExtractors[label.Name] = label.Extractor
		metricDescriptor.Labels = append(metricDescriptor.Labels, &logging.LabelDescriptor{
			Key:         label.Name,
			Description: label.Description,
			ValueType:   label.ValueType,
		})
	}
	return nil
}
