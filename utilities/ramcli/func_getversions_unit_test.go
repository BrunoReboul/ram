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

func TestUnitGetVersions(t *testing.T) {
	var testCases = []struct {
		name           string
		repositoryPath string
		wantErrorMsg   string
		wantGoVersion  string
		wantRAMVersion string
	}{
		{
			name:           "go113ram012rc04",
			repositoryPath: "testdata/getversions/go113ram012rc04",
			wantGoVersion:  "1.13",
			wantRAMVersion: "v0.1.2-rc04",
		},
		{
			name:           "noGoModFile",
			repositoryPath: "blabla",
			wantErrorMsg:   "no such file or directory",
		},
		{
			name:           "missingGoVersion",
			repositoryPath: "testdata/getversions/missinggoversion",
			wantErrorMsg:   "goVersion NOT found",
		},
		{
			name:           "missingRamVersion",
			repositoryPath: "testdata/getversions/missingramversion",
			wantErrorMsg:   "ramVersion NOT found",
		},
	}
	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			goVersion, ramVersion, err := getVersions(tc.repositoryPath)
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
					if tc.wantGoVersion != goVersion {
						t.Errorf("Want goversion %s got %s", tc.wantGoVersion, goVersion)
					}
					if tc.wantRAMVersion != ramVersion {
						t.Errorf("Want goversion %s got %s", tc.wantGoVersion, goVersion)
					}
				} else {
					t.Errorf("Expect this error did not get it %s", tc.wantErrorMsg)
				}
			}
		})
	}
}
