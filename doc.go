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
Package realtimeassetmonitor RAM Real-time Asset Monitor

## What

Audit Google Cloud resources (the assets) compliance against a set of rules when the resource is updated. The stream of detected non compliances could then be consumed to alert, report or even fix on the fly.

### Use cases

1. Security compliance, usually 80% of the rules
2. Operational compliance
   - E.g. each Cloud SQL MySQL instance should have a defined maintenance window to avoid downtime
3. Financial Operations (finOps) compliance
   - E.g. Do not provision anymore N1 virtual machines instances, instead provision N2: the price performance ratio is better

## Why

- It is all easier to fix when it is detected early
- Value is delivered only when a detected non compliance is fixed
*/
package realtimeassetmonitor
