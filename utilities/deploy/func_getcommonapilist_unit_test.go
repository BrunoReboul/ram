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

func TestGetCommonAPIlist(t *testing.T) {
	var tests = []struct {
		name                string
		wantNumberCommonAPI int
	}{
		{"ExactNumberOfCommonAPIs", 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			countCommonAPIs := len(GetCommonAPIlist())
			if test.wantNumberCommonAPI != countCommonAPIs {
				t.Errorf("Want %d common APIs got %d", test.wantNumberCommonAPI, countCommonAPIs)
			}
		})
	}
}
