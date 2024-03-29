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

import (
	"log"

	"github.com/BrunoReboul/ram/utilities/deploy"
)

// initialize intitial setup steps before manual, aka not automatable, setup tasks
func (deployment *Deployment) initialize() (err error) {
	deployment.Core.InstanceName = "initial setup"
	if err = deployment.deployGRMFolder(); err != nil {
		return err
	}
	if err = deployment.deployGRMProject(); err != nil {
		return err
	}
	if err = deployment.enableBILBillingAccountOnProject(); err != nil {
		return err
	}
	deployment.Settings.Service.GSU.APIList = deploy.GetCommonAPIlist()
	if err = deployment.deployGSUAPI(); err != nil {
		return err
	}
	log.Println("")
	log.Println("manual setup task 1: console / firestore / select native mode")
	log.Println("manual setup task 2: OPTIONAL: console / source repo / add a repo CONNECTED to external repo")
	log.Println("")
	return nil
}
