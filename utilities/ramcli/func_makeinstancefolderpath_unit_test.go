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
	"testing"
)

func TestUnitMakeInstanceFolderPath(t *testing.T) {
	var testCases = []struct {
		name                string
		instancesFolderPath string
		instanceName        string
		instanceFolderPath  string
	}{
		{
			name:                "mixture",
			instancesFolderPath: "services/mysvc/instances",
			instanceName:        "Joli mois de Mai-Joli mois de Juin-Joli mois de Juillet-Joli mois de Aout",
			instanceFolderPath:  "services/mysvc/instances/Joli mois de Mai_Joli mois de Juin_Joli mois de Juillet_Joli mo",
		},
	}
	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			instanceFolderPath := makeInstanceFolderPath(tc.instancesFolderPath, tc.instanceName)
			if instanceFolderPath != tc.instanceFolderPath {
				t.Errorf("Error want '%s' and got '%s'", tc.instanceFolderPath, instanceFolderPath)
			}
		})
	}
}
