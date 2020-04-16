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

package gcftemplate

// BackgroundPubSubTriggeredFunctionGo function.go code skeleton, replace %s by serviceName
const BackgroundPubSubTriggeredFunctionGo = `
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

// Package p contains a background cloud function
package p

import (
	"context"

	"github.com/BrunoReboul/ram/services/%s"
	"github.com/BrunoReboul/ram/utilities/ram"
)

var global %s.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
	return %s.EntryPoint(ctxEvent, PubSubMessage, &global)
}

func init() {
	%s.Initialize(ctx, &global)
}
`
