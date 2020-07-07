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

package gcs

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"cloud.google.com/go/storage"
)

// Deploy bucket
func (bucketDeployment *BucketDeployment) Deploy() (err error) {
	log.Printf("%s gcs bucket %s", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	var lifecycle storage.Lifecycle
	var lifecycleRule storage.LifecycleRule
	lifecycleRule.Action.Type = "Delete"
	lifecycleRule.Condition.AgeInDays = bucketDeployment.Settings.DeleteAgeInDays
	lifecycle.Rules = append(lifecycle.Rules, lifecycleRule)
	bucket := bucketDeployment.Core.Services.StorageClient.Bucket(bucketDeployment.Settings.BucketName)
	retreivedAttrs, err := bucket.Attrs(bucketDeployment.Core.Ctx)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			return fmt.Errorf("bucket.Attrs %v", err)
		}
		// Create
		var bucketAttrs storage.BucketAttrs
		bucketAttrs.Location = bucketDeployment.Core.SolutionSettings.Hosting.GCF.Region
		bucketAttrs.StorageClass = "STANDARD"
		bucketAttrs.Labels = map[string]string{"name": strings.ToLower(bucketDeployment.Settings.BucketName)}
		bucketAttrs.Lifecycle = lifecycle
		bucketAttrs.UniformBucketLevelAccess.Enabled = true

		err = bucket.Create(bucketDeployment.Core.Ctx, bucketDeployment.Core.SolutionSettings.Hosting.ProjectID, &bucketAttrs)
		if err != nil {
			return fmt.Errorf("bucket.Create %v", err)
		}
		log.Printf("%s gcs bucket created %s", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
		return nil
	}
	log.Printf("%s gcs bucket found %s", bucketDeployment.Core.InstanceName, retreivedAttrs.Name)
	var bucketAttrsToUpdate storage.BucketAttrsToUpdate
	toBeUpdated := false
	if retreivedAttrs.Labels != nil {
		if retreivedAttrs.Labels["name"] != strings.ToLower(bucketDeployment.Settings.BucketName) {
			toBeUpdated = true
			bucketAttrsToUpdate.SetLabel("name", strings.ToLower(bucketDeployment.Settings.BucketName))
			log.Printf("%s gcs bucket label to be updated", bucketDeployment.Core.InstanceName)
		}
	} else {
		toBeUpdated = true
		bucketAttrsToUpdate.SetLabel("name", strings.ToLower(bucketDeployment.Settings.BucketName))
		log.Printf("%s gcs bucket label to be updated", bucketDeployment.Core.InstanceName)
	}
	if retreivedAttrs.Lifecycle.Rules != nil {
		if reflect.DeepEqual(retreivedAttrs.Lifecycle.Rules, lifecycle.Rules) {
			toBeUpdated = true
			bucketAttrsToUpdate.Lifecycle = &lifecycle
			log.Printf("%s gcs bucket lifecycle to be updated", bucketDeployment.Core.InstanceName)
		}
	} else {
		toBeUpdated = true
		bucketAttrsToUpdate.Lifecycle = &lifecycle
		log.Printf("%s gcs bucket lifecycle to be updated", bucketDeployment.Core.InstanceName)
	}
	if retreivedAttrs.UniformBucketLevelAccess.Enabled != true {
		toBeUpdated = true
		bucketAttrsToUpdate.UniformBucketLevelAccess.Enabled = true
		log.Printf("%s gcs bucket uniform level access to be updated", bucketDeployment.Core.InstanceName)
	}
	if toBeUpdated {
		retreivedAttrs, err = bucket.Update(bucketDeployment.Core.Ctx, bucketAttrsToUpdate)
		if err != nil {
			return fmt.Errorf("bucket.Update %v", err)
		}
		log.Printf("%s gcs bucket attributes updated", bucketDeployment.Core.InstanceName)
	}
	return nil
}
