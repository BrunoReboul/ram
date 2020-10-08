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

package deploy

import (
	"testing"
)

func TestUnitGetCommonAPIlist(t *testing.T) {
	var testCases = []struct {
		name                string
		wantNumberCommonAPI int
	}{
		{"ExactNumberOfCommonAPIs", 11},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			countCommonAPIs := len(GetCommonAPIlist())
			if tc.wantNumberCommonAPI != countCommonAPIs {
				t.Errorf("Want %d common APIs got %d", tc.wantNumberCommonAPI, countCommonAPIs)
			}
		})
	}
}
