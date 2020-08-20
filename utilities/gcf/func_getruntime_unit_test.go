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

package gcf

import (
	"fmt"
	"testing"
)

func TestUnitGetRunTime(t *testing.T) {
	var tests = []struct {
		input, expectedOutput string
	}{
		{"1.11", "Deprecated"},
		{"1.12", "Unsupported"},
		{"1.13", "go113"},
		{"1.14", "Unsupported"},
		{"blabla", "Unsupported"},
	}

	for _, test := range tests {
		testName := fmt.Sprintf(" %s => %s", test.input, test.expectedOutput)
		t.Run(testName, func(t *testing.T) {
			result, err := getRunTime(test.input)
			if test.expectedOutput == "Unsupported" || test.expectedOutput == "Deprecated" {
				if err == nil {
					t.Errorf("Should send back an error and is NOT")
				}
			} else {
				if result != test.expectedOutput {
					t.Errorf("got %s, want %s", result, test.expectedOutput)
				}
			}
		})
	}
}
