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

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/BrunoReboul/ram/utilities/str"
	"gopkg.in/yaml.v2"
)

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

func makeConstraintsYAML(repositoryPath string, constraintFolderRelativePaths []string) (cs constraints, err error) {
	// defer elapsed(track("makeConstraintsYAML"))
	var s service
	var r rule
	var c constraint
	for _, constraintFolderRelativePath := range constraintFolderRelativePaths {
		// fmt.Println(constraintFolderRelativePath)
		err = ffo.ReadUnmarshalYAML(fmt.Sprintf("%s/%s/constraint.yaml", repositoryPath, constraintFolderRelativePath), &c)
		if err != nil {
			return cs, err
		}
		parts := strings.Split(constraintFolderRelativePath, "/")
		microserviceName := parts[3]
		parts = strings.Split(microserviceName, "_")
		pathServiceName := parts[1]
		pathRuleName := strings.Replace(microserviceName, fmt.Sprintf("%s_%s_", parts[0], parts[1]), "", 1)

		if pathServiceName != s.Name {
			// fmt.Println("\tnew service and new rule")
			if s.Name == "" {
				// fmt.Println("\t\tThe first path, nothing to harvest")
			} else {
				// fmt.Println("\t\tharvest previous rule, previous service")
				s.Rules = append(s.Rules, r)
				cs.Services = append(cs.Services, s)
			}
			// fmt.Println("\t\tinit rule, init service")
			r.Name = pathRuleName
			r.Constraints = []constraint{}
			s.Name = pathServiceName
			s.Rules = []rule{}
		} else {
			if pathRuleName != r.Name {
				// fmt.Println("\tcontinuing service new rule (rule harvest then init)")
				s.Rules = append(s.Rules, r)
				r.Name = pathRuleName
				r.Constraints = []constraint{}
			} else {
				// fmt.Println("\tcontinuing service continuing rule")
			}
		}
		r.Constraints = append(r.Constraints, c)
	}
	// fmt.Println("\t\tfinal step, harvest previous rule, previous service")
	s.Rules = append(s.Rules, r)
	cs.Services = append(cs.Services, s)

	absolutePath, err := filepath.Abs(repositoryPath)
	if err != nil {
		return cs, err
	}
	parts := strings.Split(absolutePath, "/")
	summary := fmt.Sprintln("# repo: " + parts[len(parts)-1])
	var totalRules, totalConstraints, serviceNumConstraints int
	for _, s := range cs.Services {
		serviceNumConstraints = 0
		for _, r := range s.Rules {
			serviceNumConstraints = serviceNumConstraints + len(r.Constraints)
		}
		summary = summary + fmt.Sprintf("# %-15s%-4drules %-4dconstraints\n", s.Name, len(s.Rules), serviceNumConstraints)
		totalConstraints = totalConstraints + serviceNumConstraints
		totalRules = totalRules + len(s.Rules)
	}
	summary = summary + fmt.Sprintf("# %d services %d rules %d constraints\n#\n", len(cs.Services), totalRules, totalConstraints)
	fmt.Print(summary)
	summary = fmt.Sprintf("# generated code %v\n", time.Now()) + summary

	bytes, err := yaml.Marshal(&cs)
	if err != nil {
		return cs, err
	}
	bytes = append([]byte(summary), bytes...)
	bytes = append([]byte(str.YAMLDisclaimer), bytes...)
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s/monitor/constraints.yaml", repositoryPath, solution.MicroserviceParentFolderName),
		bytes, 0644)
	if err != nil {
		return cs, err
	}

	return cs, nil
}
