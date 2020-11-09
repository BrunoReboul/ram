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
	"testing"
)

func TestUnitMakeConstraintsReadme(t *testing.T) {
	var testCases = []struct {
		name           string
		repositoryPath string
	}{
		{
			name:           "standard",
			repositoryPath: "testdata/ram_config/standard",
		},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			constraintFolderRelativePaths, err := GetConstraintFolderRelativePaths(tc.repositoryPath)
			if err != nil {
				t.Fatal(err)
			}
			cs, err := makeConstraintsYAML(tc.repositoryPath, constraintFolderRelativePaths)
			if err != nil {
				t.Fatal(err)
			}
			readme, err := makeConstraintsReadme(tc.repositoryPath, cs)
			if err != nil {
				t.Errorf("Did not expect an error an got %s", err.Error())
			} else {
				for _, s := range cs.Services {
					if !strings.Contains(readme, fmt.Sprintf("\n## %s\n", s.Name)) {
						t.Errorf("Heading L1 not found for service %s", s.Name)
					}
					for _, r := range s.Rules {
						if !strings.Contains(readme, fmt.Sprintf("\n- %s", r.Name)) {
							t.Errorf("Bullet point not found for rule %s", r.Name)
						}
					}
				}
			}
		})
	}
}
