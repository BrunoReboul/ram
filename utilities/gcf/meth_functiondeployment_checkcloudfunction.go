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

package gcf

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/BrunoReboul/ram/utilities/str"
)

// checkCloudFunction looks for and existing cloud function
func (functionDeployment *FunctionDeployment) checkCloudFunction() (err error) {
	retreivedCloudFunction, err := functionDeployment.Artifacts.ProjectsLocationsFunctionsService.Get(functionDeployment.Artifacts.CloudFunction.Name).Context(functionDeployment.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return fmt.Errorf("%s gcf function NOT found for this instance", functionDeployment.Core.InstanceName)
		}
		return fmt.Errorf("ProjectsLocationsFunctionsService.Get %v", err)
	}
	var s string
	if functionDeployment.Artifacts.CloudFunction.AvailableMemoryMb != retreivedCloudFunction.AvailableMemoryMb {
		s = fmt.Sprintf("%savailableMemoryMb\nwant %d\nhave %d\n", s,
			functionDeployment.Artifacts.CloudFunction.AvailableMemoryMb,
			retreivedCloudFunction.AvailableMemoryMb)
	}
	if functionDeployment.Artifacts.CloudFunction.Description != retreivedCloudFunction.Description {
		s = fmt.Sprintf("%sdescription\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.Description,
			retreivedCloudFunction.Description)
	}
	if functionDeployment.Artifacts.CloudFunction.EntryPoint != retreivedCloudFunction.EntryPoint {
		s = fmt.Sprintf("%sentryPoint\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.EntryPoint,
			retreivedCloudFunction.EntryPoint)
	}
	if functionDeployment.Artifacts.CloudFunction.EventTrigger.EventType != retreivedCloudFunction.EventTrigger.EventType {
		s = fmt.Sprintf("%seventTrigger.EventType\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.EventTrigger.EventType,
			retreivedCloudFunction.EventTrigger.EventType)
	}
	if functionDeployment.Artifacts.CloudFunction.EventTrigger.Resource != retreivedCloudFunction.EventTrigger.Resource {
		s = fmt.Sprintf("%seventTrigger.Resource\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.EventTrigger.Resource,
			retreivedCloudFunction.EventTrigger.Resource)
	}
	if functionDeployment.Artifacts.CloudFunction.EventTrigger.Service != retreivedCloudFunction.EventTrigger.Service {
		s = fmt.Sprintf("%seventTrigger.Service\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.EventTrigger.Service,
			retreivedCloudFunction.EventTrigger.Service)
	}
	if functionDeployment.Artifacts.CloudFunction.EventTrigger.FailurePolicy.Retry != retreivedCloudFunction.EventTrigger.FailurePolicy.Retry {
		s = fmt.Sprintf("%seventTrigger.FailurePolicy.Retry\nwant %v\nhave %v\n", s,
			functionDeployment.Artifacts.CloudFunction.EventTrigger.FailurePolicy.Retry,
			retreivedCloudFunction.EventTrigger.FailurePolicy.Retry)
	}
	if !reflect.DeepEqual(functionDeployment.Artifacts.CloudFunction.Labels, retreivedCloudFunction.Labels) {
		s = fmt.Sprintf("%slabels\nwant %s\nhave %s\n", s,
			str.FlattenMapStringString(functionDeployment.Artifacts.CloudFunction.Labels),
			str.FlattenMapStringString(retreivedCloudFunction.Labels))
	}
	if functionDeployment.Artifacts.CloudFunction.Name != retreivedCloudFunction.Name {
		s = fmt.Sprintf("%sname\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.Name,
			retreivedCloudFunction.Name)
	}
	if functionDeployment.Artifacts.CloudFunction.Runtime != retreivedCloudFunction.Runtime {
		s = fmt.Sprintf("%sruntime\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.Runtime,
			retreivedCloudFunction.Runtime)
	}
	if functionDeployment.Artifacts.CloudFunction.ServiceAccountEmail != retreivedCloudFunction.ServiceAccountEmail {
		s = fmt.Sprintf("%sserviceAccountEmail\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.ServiceAccountEmail,
			retreivedCloudFunction.ServiceAccountEmail)
	}
	if functionDeployment.Artifacts.CloudFunction.Timeout != retreivedCloudFunction.Timeout {
		s = fmt.Sprintf("%stimeout\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.Timeout,
			retreivedCloudFunction.Timeout)
	}
	if functionDeployment.Artifacts.CloudFunction.IngressSettings != retreivedCloudFunction.IngressSettings {
		s = fmt.Sprintf("%singressSettings\nwant %s\nhave %s\n", s,
			functionDeployment.Artifacts.CloudFunction.IngressSettings,
			retreivedCloudFunction.IngressSettings)
	}

	if len(s) > 0 {
		return fmt.Errorf("%s gcf invalid cloud function configuration:\n%s", functionDeployment.Core.InstanceName, s)
	}
	return nil
}
