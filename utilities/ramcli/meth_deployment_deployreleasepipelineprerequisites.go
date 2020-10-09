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

import "github.com/BrunoReboul/ram/utilities/deploy"

// deployReleasePipelinePrerequsites deploy once the prerequisites to create then cloud build triggers
func (deployment *Deployment) deployReleasePipelinePrerequsites() (err error) {
	// Common pre requisite, specific pre req (aka custom roles) are addresses in deployInstanceReleasePipeline
	// see https://github.com/BrunoReboul/ram/issues/44
	if err = deployment.enableBILBillingAccountOnProject(); err != nil {
		return err
	}
	deployment.Settings.Service.GSU.APIList = deploy.GetCommonAPIlist()
	if err = deployment.deployGSUAPI(); err != nil {
		return err
	}
	return nil
}
