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

package deploy

// GetCommonAPIlist returns the list of common APIs
func GetCommonAPIlist() []string {
	return []string{
		"bigquery.googleapis.com",
		"cloudapis.googleapis.com",
		"logging.googleapis.com",
		"monitoring.googleapis.com",
		"serviceusage.googleapis.com",
		"storage-api.googleapis.com",
		"storage-component.googleapis.com"}
}
