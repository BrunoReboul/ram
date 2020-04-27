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

package grm

import (
	"fmt"
	"log"
)

// Deploy FolderDeployment for now, only check the folder exist and is ACTIVE: It does NOT create the folder.
func (folderDeployment *FolderDeployment) Deploy() (err error) {
	log.Printf("%s grm resource manager folder", folderDeployment.Core.InstanceName)
	foldersService := folderDeployment.Core.Services.CloudresourcemanagerServicev2.Folders
	folderName := fmt.Sprintf("folders/%s", folderDeployment.Core.SolutionSettings.Hosting.FolderID)
	folder, err := foldersService.Get(folderName).Context(folderDeployment.Core.Ctx).Do()
	if err != nil {
		return err
	}
	if folder.LifecycleState != "ACTIVE" {
		return fmt.Errorf("%s grm folder %s %s is in state %s while it should be ACTIVE", folderDeployment.Core.InstanceName,
			folder.Name,
			folder.DisplayName,
			folder.LifecycleState)
	}
	log.Printf("%s grm folder found %s %s parent %s", folderDeployment.Core.InstanceName,
		folder.Name,
		folder.DisplayName,
		folder.Parent)
	return nil
}
