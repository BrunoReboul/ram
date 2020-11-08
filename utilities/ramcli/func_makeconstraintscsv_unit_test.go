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

func TestUnitMakeConstraintsCSV(t *testing.T) {
	var testCases = []struct {
		name               string
		repositoryPath     string
		wantErrorMsg       string
		wantServiceName    string
		wantRuleName       string
		wantConstraintName string
		wantSeverity       string
		wantCategory       string
		wantKind           string
		wantDescription    string
	}{
		{
			name:           "standard",
			repositoryPath: "testdata/ram_config/standard",
		},
		{
			name:               "onlyOneConstraint",
			repositoryPath:     "testdata/ram_config/onlyoneconstraint",
			wantServiceName:    "bq",
			wantRuleName:       "dataset_location",
			wantConstraintName: "myorg_sanboxes_europe_bq",
			wantSeverity:       "critical",
			wantCategory:       "Personnal Data Compliance",
			wantKind:           "GCPBigQueryDatasetLocationConstraintV1",
			wantDescription:    "BQ Dataset must be in Europe for projects in myorg/sandboxes/Europe.",
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
			numberOfPaths := len(constraintFolderRelativePaths)
			records, err := makeConstraintsCSV(tc.repositoryPath, constraintFolderRelativePaths)

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
					if numberOfPaths+1 != len(records) {
						t.Errorf("want number of records %d got %d", numberOfPaths+1, len(records))
					}
					ouputfilePath := fmt.Sprintf("%s/services/monitor/constraints.csv", tc.repositoryPath)
					_, err = os.Stat(ouputfilePath)
					if err != nil {
						t.Errorf("Did not expect an error an got %s", err.Error())
					}
					if tc.name == "onlyOneConstraint" {
						if records[1][0] != tc.wantServiceName {
							t.Errorf("ServiceName want %s got %s", tc.wantServiceName, records[1][0])
						}
						if records[1][1] != tc.wantRuleName {
							t.Errorf("RuleName want %s got %s", tc.wantRuleName, records[1][1])
						}
						if records[1][2] != tc.wantConstraintName {
							t.Errorf("ConstraintName want %s got %s", tc.wantConstraintName, records[1][2])
						}
						if records[1][3] != tc.wantSeverity {
							t.Errorf("Severity want %s got %s", tc.wantSeverity, records[1][3])
						}
						if records[1][4] != tc.wantCategory {
							t.Errorf("Category want %s got %s", tc.wantCategory, records[1][4])
						}
						if records[1][5] != tc.wantKind {
							t.Errorf("Kind want %s got %s", tc.wantKind, records[1][5])
						}
						if records[1][6] != tc.wantDescription {
							t.Errorf("Description want %s got %s", tc.wantDescription, records[1][6])
						}
					}
				} else {
					t.Errorf("Expect this error did not get it %s", tc.wantErrorMsg)
				}
			}
		})
	}
}
