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

package ramcli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BrunoReboul/ram/services/setdashboards"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

// configureSetDashboards
func (deployment *Deployment) configureSetDashboards() (err error) {
	serviceName := "setdashboards"
	serviceFolderPath := fmt.Sprintf("%s/%s/%s",
		deployment.Core.RepositoryPath,
		solution.MicroserviceParentFolderName,
		serviceName)
	if _, err := os.Stat(serviceFolderPath); os.IsNotExist(err) {
		os.Mkdir(serviceFolderPath, 0755)
	}

	log.Printf("configure %s", serviceName)
	var setDashboardsInstanceDeployment setdashboards.InstanceDeployment
	setDashboardsInstance := setDashboardsInstanceDeployment.Settings.Instance
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	// var dashboards = map[string][]string{
	// 	"RAM core microservices":   []string{"dumpinventory", "splitdump", "monitor", "stream2bq", "publish2fs", "upload2gcs"},
	// 	"RAM groups microservices": []string{"convertlog2feed", "listgroups", "getgroupsettings", "listgroupmembers"},
	// }
	// setDashboardsInstance.MON.Columns = 4
	// setDashboardsInstance.MON.WidgetTypeList = []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"}

	type dboard struct {
		columns              int64
		microServiceNameList []string
		widgetTypeList       []string
	}
	type dboards map[string]dboard

	var dashboard dboard
	var dashboards dboards
	dashboards = make(dboards)

	dashboard.columns = 4
	dashboard.widgetTypeList = []string{"widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"}
	dashboard.microServiceNameList = []string{"dumpinventory", "splitdump", "monitor", "stream2bq", "publish2fs", "upload2gcs"}
	dashboards["RAM core microservices"] = dashboard

	dashboard.microServiceNameList = []string{"convertlog2feed", "listgroups", "getgroupsettings", "listgroupmembers"}
	dashboards["RAM groups microservices"] = dashboard

	// dashboard.columns = 3
	// dashboard.widgetTypeList = []string{"widgetRAMe2eLatency", "widgetRAMLatency", "widgetRAMTriggerAge"}
	// dashboard.microServiceNameList = []string{"stream2bq", "monitor", "upload2gcs", "publish2fs", "splitdump", "dumpinventory", "listgroupmembers", "getgroupsettings", "listgroups", "convertlog2feed"}
	// dashboards["RAM latency"] = dashboard

	dashboard.columns = 1
	dashboard.widgetTypeList = []string{"widgetRAMe2eLatency", "widgetRAMLatency", "widgetRAMTriggerAge", "widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"}
	for _, microServiceName := range []string{"stream2bq", "monitor", "upload2gcs", "publish2fs", "splitdump", "dumpinventory", "listgroupmembers", "getgroupsettings", "listgroups", "convertlog2feed"} {
		dashboard.microServiceNameList = []string{microServiceName}
		dashboards[fmt.Sprintf("RAM %s", microServiceName)] = dashboard
	}

	for displayName, dashboard := range dashboards {
		setDashboardsInstance.MON.DisplayName = displayName
		setDashboardsInstance.MON.Columns = dashboard.columns
		setDashboardsInstance.MON.WidgetTypeList = dashboard.widgetTypeList
		setDashboardsInstance.MON.MicroServiceNameList = dashboard.microServiceNameList
		instanceFolderPath := fmt.Sprintf("%s/%s_%s",
			instancesFolderPath,
			serviceName,
			strings.ToLower(strings.Replace(displayName, " ", "_", -1)))
		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s",
			instanceFolderPath,
			solution.InstanceSettingsFileName),
			setDashboardsInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}
	return nil
}
