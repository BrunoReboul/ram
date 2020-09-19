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
	"testing"
)

func TestUnitBuildAncestryPath(t *testing.T) {
	var tests = []struct {
		name      string
		ancestors []string
		want      string
	}{
		{
			name: "immutable",
			ancestors: []string{
				"projects/123456789012",
				"folders/123456789012",
				"folders/123456789012",
				"organizations/123456789012",
			},
			want: "organization/123456789012/folder/123456789012/folder/123456789012/project/123456789012",
		},
		{
			name: "mutable",
			ancestors: []string{
				"my-project",
				"folder-level2",
				"folder-lovel1",
				"myorganization.org",
			},
			want: "myorganization.org/folder-lovel1/folder-level2/my-project",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildAncestryPath(test.ancestors)
			if test.want != got {
				t.Errorf("Want %s got %s", test.want, got)
			}
		})
	}
}
