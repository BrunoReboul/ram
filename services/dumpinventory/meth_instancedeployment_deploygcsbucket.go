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

package dumpinventory

import (
	"github.com/BrunoReboul/ram/utilities/gcs"
)

func (instanceDeployment *InstanceDeployment) deployGCSBucket() (err error) {
	bucketDeployment := gcs.NewBucketDeployment()
	bucketDeployment.Core = instanceDeployment.Core
	bucketDeployment.Settings.BucketName = instanceDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.Name
	if bucketDeployment.Settings.DeleteAgeInDays == 0 {
		bucketDeployment.Settings.DeleteAgeInDays = bucketDeployment.Core.SolutionSettings.Hosting.GCS.Buckets.CAIExport.DeleteAgeInDays
	}
	return bucketDeployment.Deploy()
}
