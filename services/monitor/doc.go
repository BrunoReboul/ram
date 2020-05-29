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

/*
Package monitor check asset compliance

Triggered by

Resource or IAM policies assets feed messages in PubSub topics.

Instances

- one per REGO rule.

- all constraints (yaml settings) related to a REGO rule are evaluated in that REGO rule instance.

Output

- PubSub violation topic.

- PubSub complianceStatus topic.

Cardinality

- When compliant: one-one, only the compliance state, no violations.

- When not compliant: one-few, 1 compliance state + n violations.

Automatic retrying

Yes.

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/monitorcompliance"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global monitorcompliance.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return monitorcompliance.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     monitorcompliance.Initialize(ctx, &global)
 }

*/
package monitor
