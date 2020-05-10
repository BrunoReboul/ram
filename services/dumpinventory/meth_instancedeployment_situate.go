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

package dumpinventory

import (
	"fmt"
	"strings"
)

// Situate complement settings taking in account the situation for service and instance settings
func (instanceDeployment *InstanceDeployment) Situate() (err error) {
	instanceDeployment.Settings.Service.GCF.FunctionType = "backgroundPubSub"
	parts := strings.Split(instanceDeployment.Core.InstanceName, "-")
	instanceDeployment.Settings.Service.GCF.Description = fmt.Sprintf("export %s %s assets metadata from %s to storage bucket %s",
		parts[len(parts)-2],
		parts[len(parts)-1],
		instanceDeployment.Settings.Instance.CAI.Parent,
		instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name)
	return nil
}
