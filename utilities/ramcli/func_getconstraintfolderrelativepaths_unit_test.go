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
	"strings"
	"testing"
)

func TestUnitGetConstraintFolderRelativePaths(t *testing.T) {
	var testCases = []struct {
		name              string
		repositoryPath    string
		wantErrorMsg      string
		wantNumberOfPaths int
	}{
		{
			name:              "standard",
			repositoryPath:    "testdata/ram_config/standard",
			wantNumberOfPaths: 28,
		},
		{
			name:              "onlyOneConstraint",
			repositoryPath:    "testdata/ram_config/onlyoneconstraint",
			wantNumberOfPaths: 1,
		},
		{
			name:              "noConstraint",
			repositoryPath:    "testdata/ram_config/noconstraint",
			wantNumberOfPaths: 0,
		},
		{
			name:           "empty",
			repositoryPath: "testdata/ram_config/empty",
			wantErrorMsg:   "no such file or directory",
		},
		{
			name:           "noMonitorFolder",
			repositoryPath: "testdata/ram_config/nomonitorfolder",
			wantErrorMsg:   "no such file or directory",
		},
	}
	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			constraintFolderRelativePaths, err := getConstraintFolderRelativePaths(tc.repositoryPath)
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
					if tc.wantNumberOfPaths != len(constraintFolderRelativePaths) {
						t.Errorf("wantNumberOfPaths %d got %d", tc.wantNumberOfPaths, len(constraintFolderRelativePaths))
					}
				} else {
					t.Errorf("Expect this error did not get it %s", tc.wantErrorMsg)
				}
			}
		})
	}
}
