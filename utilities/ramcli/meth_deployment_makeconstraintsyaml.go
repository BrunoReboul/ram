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
	"log"
	"sort"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

func (deployment *Deployment) makeConstraintsYAML() (err error) {
	var constraintFolderRelativePaths []string
	instanceFolderRelativePaths, err := ffo.GetChild(deployment.Core.RepositoryPath,
		fmt.Sprintf("%s/%s/%s", solution.MicroserviceParentFolderName, "monitor", solution.InstancesFolderName))
	if err != nil {
		return err
	}
	sort.Strings(instanceFolderRelativePaths)
	for _, instanceFolderRelativePath := range instanceFolderRelativePaths {
		instanceConstraintFolderRelativePaths, err := ffo.GetChild(deployment.Core.RepositoryPath,
			fmt.Sprintf("%s/constraints", instanceFolderRelativePath))
		if err != nil {
			return err
		}
		sort.Strings(instanceConstraintFolderRelativePaths)
		for _, constraintFolderRelativePath := range instanceConstraintFolderRelativePaths {
			constraintFolderRelativePaths = append(constraintFolderRelativePaths, constraintFolderRelativePath)
		}
	}

	log.Printf("monitor found %d rule constraint(s)", len(constraintFolderRelativePaths))
	for _, constraintFolderRelativePath := range constraintFolderRelativePaths {
		fmt.Println(constraintFolderRelativePath)
	}

	// ffo.ReadUnmarshalYAML()

	return nil
}
