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

package bil

import (
	"fmt"
	"log"
	"strings"

	"github.com/BrunoReboul/ram/utilities/ram"

	"google.golang.org/api/cloudbilling/v1"
)

// Enable ProjectBillingAccount checks if project has a billing account, if not enable the specified billing account
func (projectBillingAccount *ProjectBillingAccount) Enable() (err error) {
	log.Printf("%s bil check project billing account", projectBillingAccount.Core.InstanceName)
	projectsService := projectBillingAccount.Core.Services.Cloudbillingservice.Projects
	resourceName := fmt.Sprintf("projects/%s", projectBillingAccount.Core.SolutionSettings.Hosting.ProjectID)
	projectBillingInfo, err := projectsService.GetBillingInfo(resourceName).Context(projectBillingAccount.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			log.Printf("%s bil WARNING impossible to GET billing info %v", projectBillingAccount.Core.InstanceName, err)
			return nil
		}
		return fmt.Errorf("bil projectsService.GetBillingInfo(resourceName) %v", err)
	}
	log.Printf("%s project billing info retreived %s", projectBillingAccount.Core.InstanceName, resourceName)
	if projectBillingInfo.BillingEnabled {
		log.Printf("%s project billing is enable on %s", projectBillingAccount.Core.InstanceName, projectBillingInfo.BillingAccountName)
	} else {
		billingAccount := projectBillingAccount.Core.SolutionSettings.Hosting.BillingAccountID
		if billingAccount == "" {
			return fmt.Errorf("Project billing not enable and 'projectBillingAccount' settings is null string in %s", ram.SolutionSettingsFileName)
		}
		var projectBillingInfoToEnable cloudbilling.ProjectBillingInfo
		projectBillingInfoToEnable.BillingAccountName = fmt.Sprintf("billingAccounts/%s", billingAccount)
		projectBillingInfoToEnable.Name = projectBillingInfo.Name
		projectBillingInfo, err := projectsService.UpdateBillingInfo(resourceName, &projectBillingInfoToEnable).Context(projectBillingAccount.Core.Ctx).Do()
		if err != nil {
			return fmt.Errorf("err projectsService.UpdateBillingInfo %v", err)
		}
		if projectBillingInfo.BillingEnabled {
			log.Printf("%s project billing has been enabled on %s", projectBillingAccount.Core.InstanceName, projectBillingInfo.BillingAccountName)
		} else {
			return fmt.Errorf("Enabling billing account %s on project %s failed",
				projectBillingAccount.Core.SolutionSettings.Hosting.BillingAccountID,
				projectBillingAccount.Core.SolutionSettings.Hosting.ProjectID)
		}
	}
	return nil
}
