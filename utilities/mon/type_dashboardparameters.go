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

package mon

// DashboardParameters structure
type DashboardParameters struct {
	DisplayName string
	GridLayout  struct {
		Columns              int64
		MicroServiceNameList []string `yaml:"microServiceNameList,omitempty"`
		WidgetTypeList       []string `yaml:"widgetTypeList,omitempty"`
	} `yaml:"gridLayout,omitempty"`
	SLOFreshnessLayout struct {
		Columns            int64
		Origin             string
		Scope              string
		Flow               string
		SLO                float64 `yaml:"slo,omitempty"`
		CutOffBucketNumber int64   `yaml:"cutOffBucketNumber,omitempty"`
	} `yaml:"sloFreshnessLayout,omitempty"`
}
