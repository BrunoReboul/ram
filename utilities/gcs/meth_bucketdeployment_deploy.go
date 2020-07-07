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
	log.Printf("%s gcs bucket %s desired lifecyle DeleteAgeInDays %d",
		bucketDeployment.Core.InstanceName,
		bucketDeployment.Settings.BucketName,
		bucketDeployment.Settings.DeleteAgeInDays)

	var uniformBucketLevelAccess storage.UniformBucketLevelAccess
	uniformBucketLevelAccess.Enabled = true

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
		bucketAttrs.UniformBucketLevelAccess = uniformBucketLevelAccess

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
			log.Printf("%s gcs bucket %s label to be updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
		}
	} else {
		toBeUpdated = true
		bucketAttrsToUpdate.SetLabel("name", strings.ToLower(bucketDeployment.Settings.BucketName))
		log.Printf("%s gcs bucket %s label to be updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	}
	if !toBeUpdated {
		log.Printf("%s gcs bucket %s label already uptodate", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	}
	if retreivedAttrs.Lifecycle.Rules != nil {
		rules := retreivedAttrs.Lifecycle.Rules
		foundDeleteRule := false
		ruleToBeUpdated := false
		for i, rule := range rules {
			log.Printf("%s gcs bucket %s delete lifecycle analyzing rule index %d",
				bucketDeployment.Core.InstanceName,
				bucketDeployment.Settings.BucketName,
				i)
			if rule.Action.Type == "Delete" {
				foundDeleteRule = true
				if rule.Condition.AgeInDays != bucketDeployment.Settings.DeleteAgeInDays {
					ruleToBeUpdated = true
					log.Printf("%s gcs bucket %s delete lifecycle rule found age %d updated to %d",
						bucketDeployment.Core.InstanceName,
						bucketDeployment.Settings.BucketName,
						rule.Condition.AgeInDays,
						bucketDeployment.Settings.DeleteAgeInDays)
					rule.Condition.AgeInDays = bucketDeployment.Settings.DeleteAgeInDays
				}
				// Do not break, may be multiple delete rules
			}
		}
		var lifecycle storage.Lifecycle
		if foundDeleteRule {
			if ruleToBeUpdated {
				toBeUpdated = true
				lifecycle.Rules = rules
				bucketAttrsToUpdate.Lifecycle = &lifecycle
				log.Printf("%s gcs bucket %s delete lifecycle rule age to be updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
			} else {
				log.Printf("%s gcs bucket %s lifecycle already has a delete rule with the desired age set", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
			}
		} else {
			toBeUpdated = true
			rules = append(rules, lifecycleRule)
			lifecycle.Rules = rules
			bucketAttrsToUpdate.Lifecycle = &lifecycle
			log.Printf("%s gcs bucket %s lifecycle delete on age rule to be added", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
		}
	} else {
		toBeUpdated = true
		bucketAttrsToUpdate.Lifecycle.Rules = lifecycle.Rules
		log.Printf("%s gcs bucket %s has no lifecycle. to be updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	}
	if !retreivedAttrs.UniformBucketLevelAccess.Enabled {
		toBeUpdated = true
		bucketAttrsToUpdate.UniformBucketLevelAccess = &uniformBucketLevelAccess
		log.Printf("%s gcs bucket %s uniform level access to be updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	}
	if toBeUpdated {
		retreivedAttrs, err = bucket.Update(bucketDeployment.Core.Ctx, bucketAttrsToUpdate)
		if err != nil {
			return fmt.Errorf("bucket.Update %v", err)
		}
		log.Printf("%s gcs bucket %s attributes have been updated", bucketDeployment.Core.InstanceName, bucketDeployment.Settings.BucketName)
	}
	return nil
}
