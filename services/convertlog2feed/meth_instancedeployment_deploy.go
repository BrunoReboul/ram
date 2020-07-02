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

package convertlog2feed

import (
	"log"
	"time"
)

// Deploy a service instance
func (instanceDeployment *InstanceDeployment) Deploy() (err error) {
	start := time.Now()
	// Extended project
	if err = instanceDeployment.deployGSUAPI(); err != nil {
		return err
	}
	if err = instanceDeployment.deployGAEApp(); err != nil {
		return err
	}
	if err = instanceDeployment.deployIAMProjectRoles(); err != nil {
		return err
	}
	if err = instanceDeployment.deployIAMServiceAccount(); err != nil {
		return err
	}
	if err = instanceDeployment.deployGRMProjectBindings(); err != nil {
		return err
	}
	// Core project
	if err = instanceDeployment.deployGPSTopic(); err != nil {
		return err
	}
	if err = instanceDeployment.deployGCFFunction(); err != nil {
		return err
	}
	log.Printf("%s this cloud function service account needs domain wide delegation", instanceDeployment.Core.InstanceName)
	log.Printf("%s this cloud function service account needs oauth scope https://www.googleapis.com/auth/apps.groups.settings", instanceDeployment.Core.InstanceName)
	log.Printf("%s this cloud function service account needs oauth scope https://www.googleapis.com/auth/admin.directory.group.readonly", instanceDeployment.Core.InstanceName)
	log.Printf("%s done in %v minutes", instanceDeployment.Core.InstanceName, time.Since(start).Minutes())
	return nil
}
