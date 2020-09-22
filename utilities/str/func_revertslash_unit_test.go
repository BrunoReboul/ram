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
	"fmt"
	"testing"
)

func TestUnitRevertSlash(t *testing.T) {
	var testCases = []struct {
		input, output string
	}{
		{"//cloudresourcemanager.googleapis.com/projects/0123456789", "\\\\cloudresourcemanager.googleapis.com\\projects\\0123456789"},
		{"//cloudresourcemanager.googleapis.com/folders/0123456789", "\\\\cloudresourcemanager.googleapis.com\\folders\\0123456789"},
		{"//cloudresourcemanager.googleapis.com/organizations/0123456789", "\\\\cloudresourcemanager.googleapis.com\\organizations\\0123456789"},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		testName := fmt.Sprintf(" %s => %s", tc.input, tc.output)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			result := RevertSlash(tc.input)
			if result != tc.output {
				t.Errorf("got %s, want %s", result, tc.output)
			}
		})
	}
}
