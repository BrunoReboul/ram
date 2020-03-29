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

package ram

import (
	"fmt"
	"testing"
)

func TestRevertSlash(t *testing.T) {
	var tests = []struct {
		input, output string
	}{
		{"//cloudresourcemanager.googleapis.com/projects/0123456789", "\\\\cloudresourcemanager.googleapis.com\\projects\\0123456789"},
		{"//cloudresourcemanager.googleapis.com/folders/0123456789", "\\\\cloudresourcemanager.googleapis.com\\folders\\0123456789"},
		{"//cloudresourcemanager.googleapis.com/organizations/0123456789", "\\\\cloudresourcemanager.googleapis.com\\organizations\\0123456789"},
	}

	for _, test := range tests {
		testName := fmt.Sprintf(" %s => %s", test.input, test.output)
		t.Run(testName, func(t *testing.T) {
			result := RevertSlash(test.input)
			if result != test.output {
				t.Errorf("got %s, want %s", result, test.output)
			}
		})
	}
}
