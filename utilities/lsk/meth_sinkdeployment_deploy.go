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

package lsk

import (
	"fmt"
	"log"
	"strings"

	"github.com/BrunoReboul/ram/utilities/gps"

	"cloud.google.com/go/logging/logadmin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Deploy get-create-update sinks
func (sinkDeployment *SinkDeployment) Deploy() (err error) {
	log.Printf("%s lsk log sink", sinkDeployment.Core.InstanceName)
	creds, err := google.FindDefaultCredentials(sinkDeployment.Core.Ctx, "https://www.googleapis.com/auth/cloud-platform")
	logAdminClient, err := logadmin.NewClient(
		sinkDeployment.Core.Ctx,
		sinkDeployment.Settings.Instance.LSK.Parent,
		option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("logadmin.NewClient %v", err)
	}

	var sink logadmin.Sink
	sink.ID = sinkDeployment.Artifacts.SinkName
	sink.Destination = sinkDeployment.Artifacts.Destination
	sink.Filter = sinkDeployment.Settings.Instance.LSK.Filter
	sink.IncludeChildren = false

	var sinkRetreived *logadmin.Sink
	sinkFound := true
	// GET
	sinkRetreived, err = logAdminClient.Sink(sinkDeployment.Core.Ctx, sink.ID)

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			sinkFound = false
		} else {
			return fmt.Errorf("logAdminClient.Sink %v", err)
		}
	}

	if sinkDeployment.Core.Commands.Check {
		if !sinkFound {
			return fmt.Errorf("%s lsk sink NOT found for this instance", sinkDeployment.Core.InstanceName)
		}
		var s string
		if sink.Destination != sinkRetreived.Destination {
			s = fmt.Sprintf("%sdestination\nwant %s\nhave %s\n", s,
				sink.Destination,
				sinkRetreived.Destination)
		}
		if sink.Filter != sinkRetreived.Filter {
			s = fmt.Sprintf("%sfilter\nwant %s\nhave %s\n", s,
				sink.Filter,
				sinkRetreived.Filter)
		}
		if sink.IncludeChildren != sinkRetreived.IncludeChildren {
			s = fmt.Sprintf("%sincludeChildren\nwant %v\nhave %v\n", s,
				sink.IncludeChildren,
				sinkRetreived.IncludeChildren)
		}

		if err = gps.CheckTopicRole(sinkDeployment.Core.Ctx,
			sinkDeployment.Core.Services.PubsubPublisherClient,
			sinkDeployment.Artifacts.TopicFullName,
			sinkRetreived.WriterIdentity,
			"roles/pubsub.publisher"); err != nil {
			s = fmt.Sprintf("%s%s\n", s, err.Error())
		}
		if len(s) > 0 {
			return fmt.Errorf("%s lsk invalid sink configuration:\n%s", sinkDeployment.Core.InstanceName, s)
		}
		return nil
	}

	if sinkFound {
		log.Printf("%s lsk found sink %s writer identity %s", sinkDeployment.Core.InstanceName, sinkRetreived.ID, sinkRetreived.WriterIdentity)
		toUpdate := false
		if sinkRetreived.Destination != sink.Destination {
			toUpdate = true
		}
		if sinkRetreived.Filter != sink.Filter {
			toUpdate = true
		}
		if sinkRetreived.IncludeChildren != sink.IncludeChildren {
			toUpdate = true
		}
		if toUpdate {
			sinkRetreived, err = logAdminClient.UpdateSink(sinkDeployment.Core.Ctx, &sink)
			if err != nil {
				return fmt.Errorf("logAdminClient.UpdateSink %v", err)
			}
			log.Printf("%s lsk updated sink %s writer identity %s", sinkDeployment.Core.InstanceName, sinkRetreived.ID, sinkRetreived.WriterIdentity)
		}
	} else {
		sinkRetreived, err = logAdminClient.CreateSink(sinkDeployment.Core.Ctx, &sink)
		if err != nil {
			return fmt.Errorf("logAdminClient.CreateSink %v", err)
		}
		log.Printf("%s lsk created sink %s writer identity %s", sinkDeployment.Core.InstanceName, sinkRetreived.ID, sinkRetreived.WriterIdentity)
	}

	err = gps.SetTopicRole(sinkDeployment.Core.Ctx,
		sinkDeployment.Core.Services.PubsubPublisherClient,
		sinkDeployment.Artifacts.TopicFullName,
		sinkRetreived.WriterIdentity,
		"roles/pubsub.publisher")
	if err != nil {
		return fmt.Errorf("gps.SetTopicRole %v", err)
	}

	err = logAdminClient.Close()
	if err != nil {
		return fmt.Errorf("logAdminClient.Close %v", err)
	}
	return nil
}
