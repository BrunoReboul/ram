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
Package splitdump nibble large CAI dumps into PubSub asset feed messages

One dump line = one PubSub message.

Triggered by

Google Cloud Storage event when a new dump is delivered.

Instances

Only one.

Output

- PubSub messages formated like Cloud Asset Inventory real-time feed messages.

- Delivered in the same topics as used per CAI real-time.

- Tags to differentiate them from CAI real time feeds.

- Create missing topics en the fly (best effort) in case it does not already exist for real-time.

Cardinality

One-many: one dump is nubbled in many feed messages.

To ensure scallabilty the function is recurssive:

- If dump size > x lines then segments the dumo into child dumps of x lines.

- else nibble the dump, one dump line = one PubSub message.

- x is set through an environment variable, e.g. 1000.

Automatic retrying

Yes.

Is recurssive

Yes.

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/splitdump"
     "github.com/BrunoReboul/ram/utilities/gcs"
 )
 var global splitdump.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, gcsEvent gcs.Event) error {
     return splitdump.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     splitdump.Initialize(ctx, &global)
 }

Notes

- When Cloud Asset Inventory publishes an export to Cloud Storage:

- 1st Creates an empty dump object.

- 2nd Updates the objects then.

- So, two event notifications. The cloud founction ignore the first one (empty object).

*/
package splitdump
