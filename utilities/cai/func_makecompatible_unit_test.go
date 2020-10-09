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

package cai

import (
	"strings"
	"testing"
)

func TestUnitMakeCompatible(t *testing.T) {
	var testCases = []struct {
		name string
		path string
		want string
	}{
		{
			name: "NeedToReplace",
			path: "organizations/123456789012/folders/123456789012/folders/123456789012/projects/123456789012",
			want: "organization/123456789012/folder/123456789012/folder/123456789012/project/123456789012",
		},
		{
			name: "DoNothing",
			path: "organization/123456789012/folder/123456789012/folder/123456789012/project/123456789012",
			want: "organization/123456789012/folder/123456789012/folder/123456789012/project/123456789012",
		},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := makeCompatible(tc.path)
			if tc.want != got {
				t.Errorf("Want %s got %s", tc.want, got)
			}

			for _, unwanted := range []string{"organizations", "folders", "projects"} {
				if strings.Contains(got, unwanted) {
					t.Errorf("should never contain %s got %s, ", unwanted, got)
				}
			}
		})
	}
}
