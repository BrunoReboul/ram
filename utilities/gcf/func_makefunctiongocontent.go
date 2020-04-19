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
	"strings"
	"time"
)

// BackgroundPubSubFunctionGo function.go code skeleton, replace <serviceName> by serviceName
const BackgroundPubSubFunctionGo = `
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

// generated code %v

// Package p contains a background cloud function
package p

import (
	"context"

	"github.com/BrunoReboul/ram/services/<serviceName>"
	"github.com/BrunoReboul/ram/utilities/ram"
)

var global <serviceName>.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
	return <serviceName>.EntryPoint(ctxEvent, PubSubMessage, &global)
}

func init() {
	<serviceName>.Initialize(ctx, &global)
}
`

// MakeFunctionGoContent craft the content of a cloud function function.go file for a RAM microservice instance
func MakeFunctionGoContent(gcfType, serviceName string) (functionGoContent string, err error) {
	switch gcfType {
	case "backgroundPubSub":
		return fmt.Sprintf(strings.Replace(BackgroundPubSubFunctionGo, "<serviceName>", serviceName, -1), time.Now()), nil
	default:
		return "", fmt.Errorf("gcfType provided not managed: %s", gcfType)
	}
}