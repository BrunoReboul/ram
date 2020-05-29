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

package publish2fs

import (
	"time"

	"gopkg.in/yaml.v2"

	"github.com/BrunoReboul/ram/utilities/gcf"
)

func (instanceDeployment *InstanceDeployment) deployGCFFunction() (err error) {
	instanceDeployment.DumpTimestamp = time.Now()
	instanceDeploymentYAMLBytes, err := yaml.Marshal(instanceDeployment)
	if err != nil {
		return err
	}
	functionDeployment := gcf.NewFunctionDeployment()
	functionDeployment.Core = instanceDeployment.Core
	functionDeployment.Artifacts.InstanceDeploymentYAMLContent = string(instanceDeploymentYAMLBytes)
	functionDeployment.Settings.Service.GCF = instanceDeployment.Settings.Service.GCF
	functionDeployment.Settings.Instance.GCF = instanceDeployment.Settings.Instance.GCF
	return functionDeployment.Deploy()
}
