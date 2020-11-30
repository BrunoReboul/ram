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
const backgroundPubSubFunctionGo = `
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

// generated code <timeStamp>

// Package p contains a background cloud function
package p

import (
	"context"

	"github.com/BrunoReboul/ram/services/<serviceName>"
	"github.com/BrunoReboul/ram/utilities/gps"
)

var global <serviceName>.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage) error {
	return <serviceName>.EntryPoint(ctxEvent, PubSubMessage, &global)
}

func init() {
	err := <serviceName>.Initialize(ctx, &global)
	if err != nil {
		log.Fatalf("pubsub_id %s INIT_FAILURE %v", global.PubSubID, err)
	}
}
`

// BackgroundGCSFunctionGo function.go code skeleton, replace <serviceName> by serviceName
const backgroundGCSFunctionGo = `
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

// generated code <timeStamp>

// Package p contains a background cloud function
package p

import (
	"context"

	"github.com/BrunoReboul/ram/services/<serviceName>"
	"github.com/BrunoReboul/ram/utilities/gcs"
)

var global <serviceName>.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, gcsEvent gcs.Event) error {
	return <serviceName>.EntryPoint(ctxEvent, gcsEvent, &global)
}

func init() {
	err := <serviceName>.Initialize(ctx, &global)
	if err != nil {
		log.Fatalf("pubsub_id %s INIT_FAILURE %v", global.PubSubID, err)
	}
}
`

// makeFunctionGoContent craft the content of a cloud function function.go file for a RAM microservice instance
func (functionDeployment *FunctionDeployment) makeFunctionGoContent() (functionGoContent string, err error) {
	timeStamp := fmt.Sprintf("%s", time.Now())
	switch functionDeployment.Settings.Service.GCF.FunctionType {
	case "backgroundPubSub":
		return strings.Replace(strings.Replace(backgroundPubSubFunctionGo,
			"<serviceName>", functionDeployment.Core.ServiceName, -1), "<timeStamp>", timeStamp, -1), nil
	case "backgroundGCS":
		return strings.Replace(strings.Replace(backgroundGCSFunctionGo,
			"<serviceName>", functionDeployment.Core.ServiceName, -1), "<timeStamp>", timeStamp, -1), nil
	default:
		return "", fmt.Errorf("functionType provided not managed: %s", functionDeployment.Settings.Service.GCF.FunctionType)
	}
}
