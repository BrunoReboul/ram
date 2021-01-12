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
	"github.com/BrunoReboul/ram/utilities/glo"
)

func (instanceDeployment *InstanceDeployment) deploylogMetric() (err error) {
	logMetricDeployment := glo.NewLogMetricDeployment()
	logMetricDeployment.Core = instanceDeployment.Core
	logMetricDeployment.Settings.Instance.GLO = instanceDeployment.Settings.Instance.GLO
	logMetricDeployment.Artifacts.LogMetric = instanceDeployment.Artifacts.LogMetric
	return logMetricDeployment.Deploy()
}
