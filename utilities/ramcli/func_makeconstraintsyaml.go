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
	"strings"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

func makeConstraintsYAML(repositoryPath string, constraintFolderRelativePaths []string) (err error) {
	type constraint interface{}

	type rule struct {
		Name        string `yaml:"ruleName"`
		Constraints []constraint
	}

	type service struct {
		Name  string `yaml:"serviceName"`
		Rules []rule
	}

	type constraints struct {
		Services []service
	}

	var serviceName, ruleName string
	var newService, newRule bool
	var s service
	var r rule
	var c constraint
	var cs constraints
	for _, constraintFolderRelativePath := range constraintFolderRelativePaths {
		// fmt.Println(constraintFolderRelativePath)
		parts := strings.Split(constraintFolderRelativePath, "/")
		microserviceName := parts[3]
		parts = strings.Split(microserviceName, "_")
		if parts[1] != serviceName {
			serviceName = parts[1]
			newService = true
		} else {
			newService = false
		}
		rn := strings.Replace(microserviceName, fmt.Sprintf("%s_%s_", parts[0], parts[0]), "", 1)
		if rn != ruleName {
			ruleName = rn
			newRule = true
		} else {
			newRule = false
		}
		err = ffo.ReadUnmarshalYAML(fmt.Sprintf("%s/%s/constraint.yaml", repositoryPath, constraintFolderRelativePath), &c)
		if err != nil {
			return err
		}
		if newRule {
			if r.Name != "" {
				s.Rules = append(s.Rules, r)
			}
			r.Name = ruleName
			r.Constraints = []constraint{c}
		} else {
			r.Constraints = append(r.Constraints, c)
		}
		if newService {
			if s.Name != "" {
				cs.Services = append(cs.Services, s)
			}
			s.Name = serviceName
			s.Rules = []rule{r}
		} else {
			s.Rules = append(s.Rules, r)
		}
	}
	s.Rules = append(s.Rules, r)
	cs.Services = append(cs.Services, s)

	err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s/monitor/constraints.yaml", repositoryPath, solution.MicroserviceParentFolderName), &cs)
	if err != nil {
		return err
	}
	return nil
}
