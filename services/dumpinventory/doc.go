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
Package dumpinventory request CAI to perform an export

Triggered by

Cloud Scheduler Job, through PubSub messages.


Instances

- one for all IAM bindings policies.
- one per AssetType for resource metadata exports.

Output

None, CAI execute exports as an asynchonous task delivered in a Google Cloud Storage bucket.

Automatic retrying

Yes.

Required environment variables

- CAIEXPORTBUCKETNAME the name of the GCS bucket where CAI dumps should be delivered
- SETTINGSFILENAME name of the JSON setting file

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/dumpinventory"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global dumpinventory.Global
 var ctx = context.Background()
 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return dumpinventory.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     dumpinventory.Initialize(ctx, &global)
 }

*/
package dumpinventory
