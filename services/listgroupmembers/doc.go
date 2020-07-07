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
Package listgroupmembers extract all members from a group in GCI directory using the Admin SDK API

Triggered by

PubSub messages from the GCI groups topic.

Instances

Only one.

Output

PubSub messages to a dedicated topic formated like Cloud Asset Inventory feed messages.

Cardinality

One-many: one group may have many members.

There is no limit in GCI on the number of members in a group.

Automatic retrying

Yes.

Domain Wide Delegation

Yes. The service account used to run this cloud function must have domain wide delegation and the following Oauth scopes:

- https://www.googleapis.com/auth/admin.directory.group.member.readonly

Key rotation strategy

Same as listgroups microservice.

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/listgroupmembers"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global listgroupmembers.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage) error {
     return listgroupmembers.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     listgroupmembers.Initialize(ctx, &global)
 }

*/
package listgroupmembers
