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

import "encoding/json"

// GetAssetLabelValue retrieve a label value from a label key, e.g. owner of resolver contact from asset labels
func GetAssetLabelValue(labelKey string, resourceJSON json.RawMessage) (string, error) {
	var resource struct {
		Data struct {
			Labels map[string]string
		}
	}
	err := json.Unmarshal(resourceJSON, &resource)
	if err != nil {
		return "", err
	}
	if resource.Data.Labels != nil {
		if labelValue, ok := resource.Data.Labels[labelKey]; ok {
			return labelValue, nil
		}
	}
	return "", nil
}
