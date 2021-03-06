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
	instancesFolderPath := fmt.Sprintf("%s/%s", serviceFolderPath, solution.InstancesFolderName)
	if _, err := os.Stat(instancesFolderPath); os.IsNotExist(err) {
		os.Mkdir(instancesFolderPath, 0755)
	}

	var setSLOFreshnessInstanceDeployment setdashboards.InstanceDeployment
	setSLOFreshnessInstance := setSLOFreshnessInstanceDeployment.Settings.Instance
	setSLOFreshnessInstance.MON.SLOFreshnessLayout.Columns = 12
	for _, freshnessSLOdefiniton := range deployment.Core.SolutionSettings.Hosting.FreshnessSLODefinitions {
		setSLOFreshnessInstance.MON.SLOFreshnessLayout.Origin = freshnessSLOdefiniton.Origin
		switch setSLOFreshnessInstance.MON.SLOFreshnessLayout.Origin {
		case "batch-export":
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Scope = "GCP"
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Flow = "batch"
		case "real-time":
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Scope = "GCP"
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Flow = "real-time"
		case "batch-listgroups":
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Scope = "Groups"
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Flow = "batch"
		case "real-time-log-export":
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Scope = "Groups"
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Flow = "real-time"
		}
		setSLOFreshnessInstance.MON.SLOFreshnessLayout.SLO = freshnessSLOdefiniton.SLO
		setSLOFreshnessInstance.MON.SLOFreshnessLayout.CutOffBucketNumber = freshnessSLOdefiniton.CutOffBucketNumber
		setSLOFreshnessInstance.MON.DisplayName = fmt.Sprintf("SLO freshness %s %s",
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Scope,
			setSLOFreshnessInstance.MON.SLOFreshnessLayout.Flow)
		n := strings.ToLower(strings.Replace(setSLOFreshnessInstance.MON.DisplayName, " ", "_", -1))
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_%s",
			serviceName,
			n))

		if _, err := os.Stat(instanceFolderPath); os.IsNotExist(err) {
			os.Mkdir(instanceFolderPath, 0755)
		}
		if err = ffo.MarshalYAMLWrite(fmt.Sprintf("%s/%s",
			instanceFolderPath,
			solution.InstanceSettingsFileName),
			setSLOFreshnessInstance); err != nil {
			return err
		}
		log.Printf("done %s", instanceFolderPath)
	}

	var setDashboardsInstanceDeployment setdashboards.InstanceDeployment
	setDashboardsInstance := setDashboardsInstanceDeployment.Settings.Instance

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

	dashboard.columns = 3
	dashboard.widgetTypeList = []string{"widgetRAMe2eLatency", "widgetRAMLatency", "widgetRAMTriggerAge", "widgetSubOldestUnackedMsg", "widgetGCFActiveInstances", "widgetGCFExecutionCount", "widgetGCFExecutionTime", "widgetGCFMemoryUsage"}
	for _, microServiceName := range []string{"stream2bq", "monitor", "upload2gcs", "publish2fs", "splitdump", "dumpinventory", "listgroupmembers", "getgroupsettings", "listgroups", "convertlog2feed"} {
		dashboard.microServiceNameList = []string{microServiceName}
		dashboards[fmt.Sprintf("RAM %s", microServiceName)] = dashboard
	}

	for displayName, dashboard := range dashboards {
		setDashboardsInstance.MON.DisplayName = displayName
		setDashboardsInstance.MON.GridLayout.Columns = dashboard.columns
		setDashboardsInstance.MON.GridLayout.WidgetTypeList = dashboard.widgetTypeList
		setDashboardsInstance.MON.GridLayout.MicroServiceNameList = dashboard.microServiceNameList
		instanceFolderPath := makeInstanceFolderPath(instancesFolderPath, fmt.Sprintf("%s_%s",
			serviceName,
			strings.ToLower(strings.Replace(displayName, " ", "_", -1))))
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
