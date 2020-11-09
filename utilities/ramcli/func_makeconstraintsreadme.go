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
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrunoReboul/ram/utilities/solution"
	"gopkg.in/yaml.v2"
)

func makeConstraintsReadme(repositoryPath string, cs constraints) (readme string, err error) {
	absolutePath, err := filepath.Abs(repositoryPath)
	if err != nil {
		return "", err
	}
	parts := strings.Split(absolutePath, "/")
	readme = fmt.Sprintf("# Compliance rules summary\n\nRepository: **%s**\n\n*Timestamp* %v\n\nService | rules | constraints\n--- | --- | ---\n", parts[len(parts)-1], time.Now())

	var totalRules, totalConstraints, serviceNumConstraints int
	for _, s := range cs.Services {
		serviceNumConstraints = 0
		for _, r := range s.Rules {
			serviceNumConstraints = serviceNumConstraints + len(r.Constraints)
		}
		readme = readme + fmt.Sprintf("**%s** | %d | %d\n", s.Name, len(s.Rules), serviceNumConstraints)
		totalConstraints = totalConstraints + serviceNumConstraints
		totalRules = totalRules + len(s.Rules)
	}
	readme = readme + fmt.Sprintf("\n%d services %d rules %d constraints\n", len(cs.Services), totalRules, totalConstraints)

	var constraint constraintInfo
	for _, s := range cs.Services {
		readme = readme + fmt.Sprintf("\n## %s\n\n", s.Name)
		for _, r := range s.Rules {
			if len(r.Constraints) == 1 {
				readme = readme + fmt.Sprintf("- %s ", r.Name)
			} else {
				readme = readme + fmt.Sprintf("- %s\n", r.Name)
			}
			for _, c := range r.Constraints {
				bytes, err := yaml.Marshal(&c)
				if err != nil {
					return "", err
				}
				err = yaml.Unmarshal(bytes, &constraint)
				if err != nil {
					return "", err
				}
				constraintReadmeRelativePath := fmt.Sprintf("instances/monitor_%s_%s/constraints/%s/readme.md",
					s.Name,
					r.Name,
					constraint.Metadata.Name)
				readme = readme + fmt.Sprintf("  - **[%s](%s)** (*%s* %s)\n",
					constraint.Metadata.Name,
					constraintReadmeRelativePath,
					constraint.Spec.Severity,
					constraint.Metadata.Annotations.Category)
			}
		}
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s/monitor/readme.md", repositoryPath, solution.MicroserviceParentFolderName),
		[]byte(readme), 0644)
	if err != nil {
		return "", err
	}
	return readme, nil
}
