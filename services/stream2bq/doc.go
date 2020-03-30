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
Package stream2bq streams PubSub message into BigQuery tables

It can stream into 3 RAM tables: 1) assets 2) compliance states 3) violations.

Triggered by

Messages in related PubSub topics.

Instances

One per Big Query table.

- assets.

- compliance states.

- violations.

Output

Streaming into BigQuery tables.

Cardinality

One-one, one pubsub message - one stream inserted in BigQuery.

Automatic retrying

Yes.

Required environment variables

- ASSETSCOLLECTIONID the name of the FireStore collection grouping all assets documents

- BQ_DATASET name of the Big Query dataset hosting the table

- BQ_TABLE name of the Big Query table where to insert streams

- OWNERLABELKEYNAME key name for the label identifying the asset owner

- VIOLATIONRESOLVERLABELKEYNAMEkey name for the label identifying the asset violation resolver

Implementation example

 package p
 import (
     "context"

     "github.com/BrunoReboul/ram/services/stream2bq"
     "github.com/BrunoReboul/ram/utilities/ram"
 )
 var global stream2bq.Global
 var ctx = context.Background()

 // EntryPoint is the function to be executed for each cloud function occurence
 func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
     return stream2bq.EntryPoint(ctxEvent, PubSubMessage, &global)
 }

 func init() {
     stream2bq.Initialize(ctx, &global)
 }

*/
package stream2bq
