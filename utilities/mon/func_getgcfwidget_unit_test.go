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

import (
	"strings"
	"testing"
)

func TestUnitGetGCFWidget(t *testing.T) {
	tests := []struct {
		name             string
		microserviceName string
		widgetType       string
		wantErr          bool
	}{
		{
			name:             "wrongWidgetType",
			microserviceName: "blabla",
			widgetType:       "blabla",
			wantErr:          true,
		},
		{
			name:       "missingMicroserviceName",
			widgetType: "widgetGCFActiveInstances",
			wantErr:    true,
		},
		{
			name:             "widgetGCFActiveInstances",
			microserviceName: "mymicroscv",
			widgetType:       "widgetGCFActiveInstances",
			wantErr:          false,
		},
		{
			name:             "widgetGCFExecutionCount",
			microserviceName: "mymicroscv",
			widgetType:       "widgetGCFExecutionCount",
			wantErr:          false,
		},
		{
			name:             "widgetGCFExecutionTime",
			microserviceName: "mymicroscv",
			widgetType:       "widgetGCFExecutionTime",
			wantErr:          false,
		},
		{
			name:             "widgetGCFMemoryUsage",
			microserviceName: "mymicroscv",
			widgetType:       "widgetGCFMemoryUsage",
			wantErr:          false,
		},
	}
	for _, tt := range tests {
		tt := tt // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotWidget, err := GetGCFWidget(tt.microserviceName, tt.widgetType)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(err != nil) {
				if !strings.Contains(gotWidget.Title, tt.microserviceName) {
					t.Errorf("microserviceName = %s is not contained in gotWidget.Title %s", tt.microserviceName, gotWidget.Title)
				}
				for _, ds := range gotWidget.XyChart.DataSets {
					if !strings.Contains(ds.TimeSeriesQuery.TimeSeriesFilter.Filter, tt.microserviceName) {
						t.Errorf("microserviceName = %s is not contained in Filter %s", tt.microserviceName, ds.TimeSeriesQuery.TimeSeriesFilter.Filter)
					}
				}
			}
		})
	}
}
