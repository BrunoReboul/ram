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
Package upload2gcs stores feeds as JSON files in a GCS bucket

Manage file creation (with override) and deletion.

Triggered by

Messages in related PubSub topics.

Instances

one-one, one pubsub message - one file created (with override) or deleted.

Output

JSON files into a GCS bucket.

Cardinality

One-one: one feed message - one operation performed in FireStore.

Automatic retrying

Yes.

Is recurssive

Yes.

Required environment variables

- ASSETSCOLLECTIONID the name of the FireStore collection grouping all assets documents

- BUCKETNAME name of the Google Cloud Storage bucket where to write JSON files

- OWNERLABELKEYNAME key name for the label identifying the asset owner

- VIOLATIONRESOLVERLABELKEYNAME key name for the label identifying the asset violation resolver

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/upload2gcs"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global upload2gcs.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return upload2gcs.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     upload2gcs.Initialize(ctx, &global)
 }

*/
package upload2gcs
