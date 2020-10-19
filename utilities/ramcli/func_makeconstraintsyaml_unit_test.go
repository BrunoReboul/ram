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
	"os"
	"strings"
	"testing"
)

func TestUnitMakeConstraintsYAML(t *testing.T) {
	var testCases = []struct {
		name                    string
		repositoryPath          string
		wantErrorMsg            string
		wantNumberOfServices    int
		wantNumberOfRules       int
		wantNumberOfConstraints int
	}{
		{
			name:                    "standard",
			repositoryPath:          "testdata/ram_config/standard",
			wantNumberOfServices:    7,
			wantNumberOfRules:       25,
			wantNumberOfConstraints: 28,
		},
		{
			name:                    "onlyOneConstraint",
			repositoryPath:          "testdata/ram_config/onlyoneconstraint",
			wantNumberOfServices:    1,
			wantNumberOfRules:       1,
			wantNumberOfConstraints: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			constraintFolderRelativePaths, err := getConstraintFolderRelativePaths(tc.repositoryPath)
			if err != nil {
				t.Fatal(err)
			}
			cs, err := makeConstraintsYAML(tc.repositoryPath, constraintFolderRelativePaths)

			if err != nil {
				if tc.wantErrorMsg == "" {
					t.Errorf("Did not expect an error an got %s", err.Error())
				} else {
					if !strings.Contains(err.Error(), tc.wantErrorMsg) {
						t.Errorf("Error message should contains '%s' and is", tc.wantErrorMsg)
						t.Log(string('\n') + err.Error())
					}
				}
			} else {
				if tc.wantErrorMsg == "" {
					var totalRules, totalConstraints, serviceNumConstraints int
					for _, s := range cs.Services {
						serviceNumConstraints = 0
						for _, r := range s.Rules {
							serviceNumConstraints = serviceNumConstraints + len(r.Constraints)
						}
						totalConstraints = totalConstraints + serviceNumConstraints
						totalRules = totalRules + len(s.Rules)
					}
					if tc.wantNumberOfServices != len(cs.Services) {
						t.Errorf("wantNumberOfServices %d got %d", tc.wantNumberOfServices, len(cs.Services))
					}
					if tc.wantNumberOfRules != totalRules {
						t.Errorf("wantNumberOfRules %d got %d", tc.wantNumberOfRules, totalRules)
					}
					if tc.wantNumberOfConstraints != totalConstraints {
						t.Errorf("wantNumberOfConstraints %d got %d", tc.wantNumberOfConstraints, totalConstraints)
					}
					ouputfilePath := fmt.Sprintf("%s/services/monitor/constraints.yaml", tc.repositoryPath)
					_, err = os.Stat(ouputfilePath)
					if err != nil {
						t.Errorf("Did not expect an error an got %s", err.Error())
					}
				} else {
					t.Errorf("Expect this error did not get it %s", tc.wantErrorMsg)
				}
			}
		})
	}
}
