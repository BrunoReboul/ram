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
Package publish2fs publish assets resource feeds as FireStore documents

It manages creation, updates and delete.

Triggered by

Resource or IAM policies assets feed messages in PubSub topics.

Instances

- one per asset type to be persisted in FireStore.

- ussually 3: organizations, folders and projects.

Output

FireStore documents created, updated, deleted.

Cardinality

One-one, one feed message - one operation performed in FireStore

Automatic retrying

Yes.

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/publish2fs"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global publish2fs.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return publish2fs.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     publish2fs.Initialize(ctx, &global)
 }

Notes

- It replaces / by \ in asset names not to confilct with Firestore collection/document structure.

- Cloud FireStore share the same project's default location than Cloud Storage and App Engine.

- https://cloud.google.com/firestore/docs/locations#default-cloud-location

*/
package publish2fs
