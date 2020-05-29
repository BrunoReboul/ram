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
Package setfeeds set Cloud Asset Inventory feeds at organization level

Instances per targeted GCP organization, per environment

- one feed for all iam policies

- one feed per asset type for resource metadata.

Output

Cloud Asset Inventory feeds set up.


Notes

- The default quota is 100 feeds at organization level.

- It may be too small to accomodate multiple environments.

- Request additional quotas when needed.

*/
package setfeeds
