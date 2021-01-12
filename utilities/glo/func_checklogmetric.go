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
	"reflect"

	"google.golang.org/api/logging/v2"
)

func checkLogMetric(logMetric, retrievedLogMetric *logging.LogMetric) (err error) {
	var s string
	if logMetric.Description != retrievedLogMetric.Description {
		s = fmt.Sprintf("%sdescrption\nwant %s\nhave %s\n", s,
			logMetric.Description,
			retrievedLogMetric.Description)
	}
	if logMetric.Filter != retrievedLogMetric.Filter {
		s = fmt.Sprintf("%sfilter\nwant %s\nhave %s\n", s,
			logMetric.Filter,
			retrievedLogMetric.Filter)
	}
	if logMetric.ValueExtractor != retrievedLogMetric.ValueExtractor {
		s = fmt.Sprintf("%svalueExtractor\nwant %s\nhave %s\n", s,
			logMetric.ValueExtractor,
			retrievedLogMetric.ValueExtractor)
	}
	if logMetric.LabelExtractors != nil {
		if retrievedLogMetric.LabelExtractors != nil {
			if !reflect.DeepEqual(logMetric.LabelExtractors, retrievedLogMetric.LabelExtractors) {
				s = fmt.Sprintf("%sLabelExtractors\nwant %v\nhave %v\n", s,
					logMetric.LabelExtractors,
					retrievedLogMetric.LabelExtractors)
			}
		} else {
			s = fmt.Sprintf("%snot found retrievedLogMetric.LabelExtractors\n", s)
		}
	}
	if logMetric.BucketOptions != nil {
		if retrievedLogMetric.BucketOptions != nil {

			if logMetric.BucketOptions.ExplicitBuckets != nil {
				if retrievedLogMetric.BucketOptions.ExplicitBuckets != nil {
					if !reflect.DeepEqual(logMetric.BucketOptions.ExplicitBuckets.Bounds, retrievedLogMetric.BucketOptions.ExplicitBuckets.Bounds) {
						s = fmt.Sprintf("%sexplicitBucket.Bounds\nwant %v\nhave %v\n", s,
							logMetric.BucketOptions.ExplicitBuckets.Bounds,
							retrievedLogMetric.BucketOptions.ExplicitBuckets.Bounds)
					}
				} else {
					s = fmt.Sprintf("%snot found retrievedLogMetric.BucketOptions.ExplicitBuckets\n", s)
				}
			} else {
				if logMetric.BucketOptions.ExponentialBuckets != nil {
					if retrievedLogMetric.BucketOptions.ExponentialBuckets != nil {
						if logMetric.BucketOptions.ExponentialBuckets.GrowthFactor != retrievedLogMetric.BucketOptions.ExponentialBuckets.GrowthFactor {
							s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
								logMetric.BucketOptions.ExponentialBuckets.GrowthFactor,
								retrievedLogMetric.BucketOptions.ExponentialBuckets.GrowthFactor)
						}
						if logMetric.BucketOptions.ExponentialBuckets.NumFiniteBuckets != retrievedLogMetric.BucketOptions.ExponentialBuckets.NumFiniteBuckets {
							s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
								logMetric.BucketOptions.ExponentialBuckets.NumFiniteBuckets,
								retrievedLogMetric.BucketOptions.ExponentialBuckets.NumFiniteBuckets)
						}
						if logMetric.BucketOptions.ExponentialBuckets.Scale != retrievedLogMetric.BucketOptions.ExponentialBuckets.Scale {
							s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
								logMetric.BucketOptions.ExponentialBuckets.Scale,
								retrievedLogMetric.BucketOptions.ExponentialBuckets.Scale)
						}
					} else {
						s = fmt.Sprintf("%snot found retrievedLogMetric.BucketOptions.ExponentialBuckets\n", s)
					}
				} else {
					if logMetric.BucketOptions.LinearBuckets != nil {
						if retrievedLogMetric.BucketOptions.LinearBuckets != nil {
							if logMetric.BucketOptions.LinearBuckets.NumFiniteBuckets != retrievedLogMetric.BucketOptions.LinearBuckets.NumFiniteBuckets {
								s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
									logMetric.BucketOptions.LinearBuckets.NumFiniteBuckets,
									retrievedLogMetric.BucketOptions.LinearBuckets.NumFiniteBuckets)
							}
							if logMetric.BucketOptions.LinearBuckets.Offset != retrievedLogMetric.BucketOptions.LinearBuckets.Offset {
								s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
									logMetric.BucketOptions.LinearBuckets.Offset,
									retrievedLogMetric.BucketOptions.LinearBuckets.Offset)
							}
							if logMetric.BucketOptions.LinearBuckets.Width != retrievedLogMetric.BucketOptions.LinearBuckets.Width {
								s = fmt.Sprintf("%sExponentialBuckets.GrowthFactor\nwant %v\nhave %v\n", s,
									logMetric.BucketOptions.LinearBuckets.Width,
									retrievedLogMetric.BucketOptions.LinearBuckets.Width)
							}
						} else {
							s = fmt.Sprintf("%snot found retrievedLogMetric.BucketOptions.LinearBuckets\n", s)
						}
					}
				}
			}
		} else {
			s = fmt.Sprintf("%snot found retrievedLogMetric.BucketOptions\n", s)
		}
	}
	if logMetric.MetricDescriptor != nil {
		if retrievedLogMetric.MetricDescriptor != nil {
			if logMetric.MetricDescriptor.Type != retrievedLogMetric.MetricDescriptor.Type {
				s = fmt.Sprintf("%sMetricDescriptor.Type\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.Type,
					retrievedLogMetric.MetricDescriptor.Type)
			}
			if logMetric.MetricDescriptor.Name != retrievedLogMetric.MetricDescriptor.Name {
				s = fmt.Sprintf("%sMetricDescriptor.Name\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.Name,
					retrievedLogMetric.MetricDescriptor.Name)
			}
			if logMetric.MetricDescriptor.Description != retrievedLogMetric.MetricDescriptor.Description {
				s = fmt.Sprintf("%sMetricDescriptor.Description\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.Description,
					retrievedLogMetric.MetricDescriptor.Description)
			}
			if logMetric.MetricDescriptor.LaunchStage != retrievedLogMetric.MetricDescriptor.LaunchStage {
				s = fmt.Sprintf("%sMetricDescriptor.LaunchStage\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.LaunchStage,
					retrievedLogMetric.MetricDescriptor.LaunchStage)
			}
			if logMetric.MetricDescriptor.MetricKind != retrievedLogMetric.MetricDescriptor.MetricKind {
				s = fmt.Sprintf("%sMetricDescriptor.MetricKind\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.MetricKind,
					retrievedLogMetric.MetricDescriptor.MetricKind)
			}
			if logMetric.MetricDescriptor.Unit != retrievedLogMetric.MetricDescriptor.Unit {
				s = fmt.Sprintf("%sMetricDescriptor.Unit\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.Unit,
					retrievedLogMetric.MetricDescriptor.Unit)
			}
			if logMetric.MetricDescriptor.ValueType != retrievedLogMetric.MetricDescriptor.ValueType {
				s = fmt.Sprintf("%sMetricDescriptor.ValueType\nwant %s\nhave %s\n", s,
					logMetric.MetricDescriptor.ValueType,
					retrievedLogMetric.MetricDescriptor.ValueType)
			}
			if logMetric.MetricDescriptor.Labels != nil {
				if retrievedLogMetric.MetricDescriptor.Labels != nil {
					for i := range logMetric.MetricDescriptor.Labels {
						found := false
						for j := range retrievedLogMetric.MetricDescriptor.Labels {
							if logMetric.MetricDescriptor.Labels[i].Key == retrievedLogMetric.MetricDescriptor.Labels[j].Key {
								found = true
								if logMetric.MetricDescriptor.Labels[i].Description != retrievedLogMetric.MetricDescriptor.Labels[j].Description {
									s = fmt.Sprintf("%slogMetric.MetricDescriptor.Labels[i].Description\nwant %s\nhave %s\n", s,
										logMetric.MetricDescriptor.Labels[i].Description,
										retrievedLogMetric.MetricDescriptor.Labels[j].Description)
								}
								break
							}
						}
						if !found {
							s = fmt.Sprintf("%snot found logMetric.MetricDescriptor.Labels[i].Key %s\n", s,
								logMetric.MetricDescriptor.Labels[i].Key)
						}
					}
				} else {
					s = fmt.Sprintf("%snot found retrievedLogMetric.MetricDescriptor.Labels\n", s)
				}
			}
		} else {
			s = fmt.Sprintf("%snot found retrievedLogMetric.MetricDescriptor\n", s)
		}
	}
	if len(s) > 0 {
		return fmt.Errorf("glo invalid log based metric configuration:\n%s", s)
	}
	return nil
}
