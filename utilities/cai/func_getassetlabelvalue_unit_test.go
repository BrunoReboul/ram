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

func TestUnitGetAssetLabelValue(t *testing.T) {
	var tests = []struct {
		name           string
		resourceJSON   string
		labelKey       string
		wantlabelValue string
		wantError      bool
	}{
		{
			name:           "jsonError",
			wantlabelValue: "",
			wantError:      true,
		},
		{
			name:           "noLabelAtAll",
			resourceJSON:   `{}`,
			wantlabelValue: "",
			wantError:      false,
		},
		{
			name:           "notThisLabel",
			resourceJSON:   `{"data": {"labels": {"application_name": "ram"}}}`,
			labelKey:       "owner",
			wantlabelValue: "",
			wantError:      false,
		},
		{
			name:           "owner",
			resourceJSON:   `{"data": {"labels": {"owner": "cpasmoi"}}}`,
			labelKey:       "owner",
			wantlabelValue: "cpasmoi",
			wantError:      false,
		},
		{
			name:           "violationResolver",
			resourceJSON:   `{"data": {"labels": {"owner": "cpasmoi","violation_resolver": "ohnonono"}}}`,
			labelKey:       "violation_resolver",
			wantlabelValue: "ohnonono",
			wantError:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetAssetLabelValue(test.labelKey, []byte(test.resourceJSON))
			if test.wantlabelValue != got {
				t.Errorf("Want %s got %s", test.wantlabelValue, got)
			}
			if err != nil {
				if !test.wantError {
					t.Errorf("Want no error and got %v", err)
				}
			} else {
				if test.wantError {
					t.Errorf("Want an error and got no error")
				}
			}
		})
	}
}
