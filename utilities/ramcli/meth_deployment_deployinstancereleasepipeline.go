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

// deployInstanceReleasePipeline deploys the cloud build trigger related to an instance
func (deployment *Deployment) deployInstanceReleasePipeline() (err error) {
	if !deployment.Core.Commands.Check {
		// Deploy prequequsites only when not in check mode
		if err = deployment.deployGSUAPI(); err != nil {
			return err
		}
		// Extended hosting org
		if err = deployment.deployIAMHostingOrgRole(); err != nil {
			return err
		}
		if err = deployment.deployGRMHostingOrgBindings(); err != nil {
			return err
		}
		// Extended monitoring org
		if err = deployment.deployIAMMonitoringOrgRole(); err != nil {
			return err
		}
		if err = deployment.deployGRMMonitoringOrgBindings(); err != nil {
			return err
		}
		// Extended folder
		if err = deployment.deployGRMFolder(); err != nil {
			return err
		}
		if err = deployment.deployGRMProject(); err != nil {
			return err
		}
		if err = deployment.deployIAMProjectRoles(); err != nil {
			return err
		}
		if err = deployment.deployGRMProjectBindings(); err != nil {
			return err
		}
		if err = deployment.deployIAMServiceAccount(); err != nil {
			return err
		}
		if err = deployment.deployIAMBindings(); err != nil {
			return err
		}
		// Core folder
		if err = deployment.deployGSRRepo(); err != nil {
			return err
		}
	}
	if err = deployment.deployGCBTrigger(); err != nil {
		return err
	}
	return nil
}
