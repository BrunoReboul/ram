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
	"log"
	"testing"

	"github.com/BrunoReboul/ram/utilities/itst"
)

func TestIntegDashboardDeployment_Deploy(t *testing.T) {
	log.Println(itst.GetIntegrationTestsProjectID())
	// type fields struct {
	// 	Core     *deploy.Core
	// 	Settings struct {
	// 		DisplayName string
	// 		Columns     int64
	// 		Widgets     []*monitoring.Widget
	// 	}
	// }
	// tests := []struct {
	// 	name    string
	// 	fields  fields
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		dashboardDeployment := DashboardDeployment{
	// 			Core:     tt.fields.Core,
	// 			Settings: tt.fields.Settings,
	// 		}
	// 		if err := dashboardDeployment.Deploy(); (err != nil) != tt.wantErr {
	// 			t.Errorf("DashboardDeployment.Deploy() error = %v, wantErr %v", err, tt.wantErr)
	// 		}
	// 	})
	// }
}
