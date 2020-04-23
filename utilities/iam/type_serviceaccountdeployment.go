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

package iam

import (
	"github.com/BrunoReboul/ram/utilities/deploy"
	"google.golang.org/api/iam/v1"
)

// ServiceaccountDeployment struct
type ServiceaccountDeployment struct {
	Artifacts struct {
		IAMService *iam.Service `yaml:"-"`
	}
	Core deploy.Core
}

// NewServiceaccountDeployment create deployment structure
func NewServiceaccountDeployment() *ServiceaccountDeployment {
	return &ServiceaccountDeployment{}
}
