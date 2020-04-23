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

package publish2fs

import (
	"time"

	"github.com/BrunoReboul/ram/utilities/deploy"
	"github.com/BrunoReboul/ram/utilities/gcb"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gsu"
	"github.com/BrunoReboul/ram/utilities/ram"
)

const (
	description  = "publish %s assets resource feeds as FireStore documents in collection %s"
	eventService = "pubsub.googleapis.com"
	eventType    = "google.pubsub.topic.publish"
	gcfType      = "backgroundPubSub"
	serviceName  = "publish2fs"
)

// InstanceDeployment settings and artifacts structure
type InstanceDeployment struct {
	Artifacts struct {
		DumpTimestamp time.Time
	}
	Core     *deploy.Core
	Settings Settings
}

// Settings flat settings structure: solution - service - instance
type Settings struct {
	Solution ram.SolutionSettings
	Service  ServiceSettings
	Instance InstanceSettings
}

// ServiceSettings defines service settings common to all service instances
type ServiceSettings struct {
	GSU gsu.Parameters
	GCB gcb.Parameters
	GCF gcf.Parameters
}

// InstanceSettings instance specific settings
type InstanceSettings struct {
	GCF struct {
		TriggerTopic string `yaml:"triggerTopic"`
	}
}

// NewInstanceDeployment create deployment structure
func NewInstanceDeployment() *InstanceDeployment {
	return &InstanceDeployment{}
}
