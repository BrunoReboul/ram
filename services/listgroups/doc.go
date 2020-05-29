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
Package listgroups extract all groups from a GCI directory using the Admin SDK API

Triggered by

Cloud Scheduler Job, through PubSub messages.

Instances

few, one per directory customer ID.

Output

PubSub messages to a dedicated topic formated like Cloud Asset Inventory feed messages.

Cardinality

- one-several: one extraction job is scalled into x queries.

- x = (number of domains in GCI directory) x (36 email prefixes).

- email prefixes: a..z 0..9.

Automatic retrying

Yes.

Is recurssive

Yes.

Domain Wide Delegation

Yes. The service account used to run this cloud function must have domain wide delegation and the following Oauth scopes:

- https://www.googleapis.com/auth/admin.directory.group.readonly

- https://www.googleapis.com/auth/admin.directory.domain.readonly

Key rotation strategy

- A new service account key is created during the cloud function deployment in Cloud Build.

- The json key file is available to the cloud function as a local source file and is not persisted in git.

- The cloud function init function deletes any key but the current one.

- So, how to rotate service accout key? just redeploy the cloud function.

GCI authentication notes

- Read the service account json key file created during the cloud function deployment.

- Get a google jwt JSON Web token configuration from: Key JSON file, Scopes, GCI User to impersonate, aka subject, aka the super admin.

- Get an HTTP client from the jwtConfig.

- Get a clientOption from the HTTP client.

- Get a service from admin directory package fron the client option.

GCI request notes

- "my_customer": As an account administrator, you can also use the my_customer alias to represent your account's customerId.

- Prefer to use the "directory_customer_id" instead of "my_customer" to narrow the execution time of the function in case of multiple directories, e.g. sandboxes / managed

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/listgroups"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global listgroups.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return listgroups.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     listgroups.Initialize(ctx, &global)
 }

*/
package listgroups
