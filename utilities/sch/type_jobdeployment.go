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

package sch

import (
	"github.com/BrunoReboul/ram/utilities/deploy"
)

// JobDeployment struct
type JobDeployment struct {
	Core      *deploy.Core
	Artifacts struct {
		JobName   string `yaml:"jobName"`
		TopicName string `yaml:"topicName"`
		Schedule  string
	}
}

// NewJobDeployment create deployment structure
func NewJobDeployment() *JobDeployment {
	return &JobDeployment{}
}
