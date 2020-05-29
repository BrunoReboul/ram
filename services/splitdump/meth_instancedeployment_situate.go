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

package splitdump

import (
	"fmt"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	instanceDeployment.Settings.Service.GCF.FunctionType = "backgroundGCS"
	instanceDeployment.Settings.Service.GCF.Description = fmt.Sprintf("split cai exports larger than %d lines in child dumps of %d lines in the same gcs bucket %s",
		instanceDeployment.Settings.Instance.SplitThresholdLineNumber,
		instanceDeployment.Settings.Instance.SplitThresholdLineNumber,
		instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name)
	return nil
}
