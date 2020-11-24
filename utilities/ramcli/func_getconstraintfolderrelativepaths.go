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
	"fmt"
	"sort"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// GetConstraintFolderRelativePaths make the list of constraint paths
func GetConstraintFolderRelativePaths(repositoryPath string) (constraintFolderRelativePaths []string, err error) {
	instanceFolderRelativePaths, err := ffo.GetChild(repositoryPath,
		fmt.Sprintf("%s/%s/%s", solution.MicroserviceParentFolderName, "monitor", solution.InstancesFolderName))
	if err != nil {
		return constraintFolderRelativePaths, err
	}
	sort.Strings(instanceFolderRelativePaths)
	for _, instanceFolderRelativePath := range instanceFolderRelativePaths {
		instanceConstraintFolderRelativePaths, err := ffo.GetChild(repositoryPath,
			fmt.Sprintf("%s/constraints", instanceFolderRelativePath))
		if err != nil {
			return constraintFolderRelativePaths, err
		}
		sort.Strings(instanceConstraintFolderRelativePaths)
		for _, constraintFolderRelativePath := range instanceConstraintFolderRelativePaths {
			constraintFolderRelativePaths = append(constraintFolderRelativePaths, constraintFolderRelativePath)
		}
	}
	return constraintFolderRelativePaths, nil
}
