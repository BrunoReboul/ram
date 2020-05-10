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

package sch

import (
	"fmt"
	"log"
	"strings"

	schedulerpb "google.golang.org/genproto/googleapis/cloud/scheduler/v1"
)

// Deploy scheduler job
func (jobDeployment *JobDeployment) Deploy() (err error) {
	log.Printf("%s cloud scheduler job", jobDeployment.Core.InstanceName)
	name := fmt.Sprintf("projects/%s/locations/%s/jobs/%s",
		jobDeployment.Core.SolutionSettings.Hosting.ProjectID,
		jobDeployment.Core.SolutionSettings.Hosting.GCF.Region,
		jobDeployment.Settings.SCH.JobName)
	var getJobRequest schedulerpb.GetJobRequest
	getJobRequest.Name = name
	retreivedJob, err := jobDeployment.Core.Services.CloudSchedulerClient.GetJob(jobDeployment.Core.Ctx, &getJobRequest)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var pubsubTarget schedulerpb.PubsubTarget
			pubsubTarget.TopicName = fmt.Sprintf("projects/%s/topics/%s",
				jobDeployment.Core.SolutionSettings.Hosting.ProjectID,
				jobDeployment.Settings.SCH.TopicName)
			pubsubTarget.Data = []byte(fmt.Sprintf("cron schedule %s", jobDeployment.Settings.SCH.Schedule))

			var jobPubsubTarget schedulerpb.Job_PubsubTarget
			jobPubsubTarget.PubsubTarget = &pubsubTarget

			var job schedulerpb.Job
			job.Name = name
			job.Description = "Real-time Asset Monitor"
			job.Target = &jobPubsubTarget
			job.Schedule = jobDeployment.Settings.SCH.Schedule

			var createJobRequest schedulerpb.CreateJobRequest
			createJobRequest.Parent = fmt.Sprintf("projects/%s/locations/%s",
				jobDeployment.Core.SolutionSettings.Hosting.ProjectID,
				jobDeployment.Core.SolutionSettings.Hosting.GCF.Region)
			createJobRequest.Job = &job

			retreivedJob, err := jobDeployment.Core.Services.CloudSchedulerClient.CreateJob(jobDeployment.Core.Ctx, &createJobRequest)
			if err != nil {
				return fmt.Errorf("CloudSchedulerClient.CreateJob %v", err)
			}
			log.Printf("%s cloud scheduler job created %s", jobDeployment.Core.InstanceName, retreivedJob.Name)
			return nil
		}
		return fmt.Errorf("CloudSchedulerClient.GetJob %v", err)
	}
	log.Printf("%s cloud scheduler job found %s", jobDeployment.Core.InstanceName, retreivedJob.Name)
	return nil
}
