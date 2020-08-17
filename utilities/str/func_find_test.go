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

package str

import (
	"testing"
)

func TestFind(t *testing.T) {
	var tests = []struct {
		name       string
		slice      []string
		val        string
		shouldPass bool
	}{
		{
			name: "FindStringInSlice",
			slice: []string{
				"Alala", "Btata", "Csa",
			},
			val:        "Csa",
			shouldPass: true,
		},
		{
			name: "DoNotFindStringInSlice",
			slice: []string{
				"Alala", "Btata", "Csa",
			},
			val:        "QuiCa",
			shouldPass: false,
		},
		{
			name:       "DoNotFindStringInEmplySlice",
			slice:      []string{},
			val:        "QuiCa",
			shouldPass: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Find(test.slice, test.val)
			if test.shouldPass {
				if test.shouldPass != result {
					t.Errorf("Should find string '%s' ins slice %v", test.val, test.slice)
				}
			} else {
				if test.shouldPass != result {
					t.Errorf("Should NOT find string '%s' ins slice %v", test.val, test.slice)
				}
			}
		})
	}
}
