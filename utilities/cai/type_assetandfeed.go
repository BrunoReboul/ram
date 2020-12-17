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

import (
	"encoding/json"
	"time"

	"github.com/BrunoReboul/ram/utilities/logging"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/groupssettings/v1"
)

// assetGroup CAI like format
type assetGroup struct {
	Name         string          `json:"name"`
	AssetType    string          `json:"assetType"`
	Ancestors    []string        `json:"ancestors"`
	AncestryPath string          `json:"ancestryPath"`
	IamPolicy    json.RawMessage `json:"iamPolicy"`
	Resource     *admin.Group    `json:"resource"`
}

// assetGroupSettings CAI like format
type assetGroupSettings struct {
	Name      string                 `json:"name"`
	AssetType string                 `json:"assetType"`
	Ancestors []string               `json:"ancestors"`
	IamPolicy json.RawMessage        `json:"iamPolicy"`
	Resource  *groupssettings.Groups `json:"resource"`
}

// assetMember CAI like format
type assetMember struct {
	Name         string          `json:"name"`
	AssetType    string          `json:"assetType"`
	Ancestors    []string        `json:"ancestors"`
	AncestryPath string          `json:"ancestryPath"`
	IamPolicy    json.RawMessage `json:"iamPolicy"`
	Resource     member          `json:"resource"`
}

// member is sligthly different from admim.Member to have both group email and member email
type member struct {
	MemberEmail string `json:"memberEmail"`
	GroupEmail  string `json:"groupEmail"`
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Role        string `json:"role"`
	Type        string `json:"type"`
}

// FeedMessageGroup CAI like format
type FeedMessageGroup struct {
	Asset     assetGroup    `json:"asset"`
	Window    Window        `json:"window"`
	Deleted   bool          `json:"deleted"`
	Origin    string        `json:"origin"`
	StepStack logging.Steps `json:"step_stack,omitempty"`
}

// FeedMessageGroupSettings CAI like format
type FeedMessageGroupSettings struct {
	Asset     assetGroupSettings `json:"asset"`
	Window    Window             `json:"window"`
	Deleted   bool               `json:"deleted"`
	Origin    string             `json:"origin"`
	StepStack logging.Steps      `json:"step_stack,omitempty"`
}

// FeedMessageMember CAI like format
type FeedMessageMember struct {
	Asset     assetMember   `json:"asset"`
	Window    Window        `json:"window"`
	Deleted   bool          `json:"deleted"`
	Origin    string        `json:"origin"`
	StepStack logging.Steps `json:"step_stack,omitempty"`
}

// Window Cloud Asset Inventory feed message time window
type Window struct {
	StartTime time.Time `json:"startTime" firestore:"startTime"`
}
