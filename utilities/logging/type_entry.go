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

package logging

import (
	"encoding/json"
	"log"
	"time"
)

// Entry defines a Google Cloud logging structured entry
// https://cloud.google.com/logging/docs/agent/configuration#special-fields
type Entry struct {
	MicroserviceName           string    `json:"microservice_name"`
	InstanceName               string    `json:"instance_name"`
	Environment                string    `json:"environment"`
	Severity                   string    `json:"severity,omitempty"`
	Message                    string    `json:"message"`
	Description                string    `json:"description"`
	Now                        time.Time `json:"now,omitempty"`
	Trace                      string    `json:"logging.googleapis.com/trace,omitempty"`
	Component                  string    `json:"component,omitempty"`
	TriggeringPubsubID         string    `json:"triggering_pubsub_id,omitempty"`
	TriggeringPubsubTimestamp  time.Time `json:"triggering_pubsub_timestamp,omitempty"`
	TriggeringPubsubAgeSeconds float64   `json:"triggering_pubsub_age_seconds,omitempty"`
	OriginEventID              string    `json:"origin_event_id,omitempty"`
	OriginEventTimestamp       time.Time `json:"origin_event_timestamp,omitempty"`
	LatencySeconds             float64   `json:"latency_seconds,omitempty"`
	LatencyE2ESeconds          float64   `json:"latency_e2e_seconds,omitempty"`
}

// String renders an entry structure to the JSON format expected by Cloud Logging.
func (e Entry) String() string {
	if e.Severity == "" {
		e.Severity = "INFO"
	}
	out, err := json.Marshal(e)
	if err != nil {
		log.Printf("json.Marshal: %v", err)
	}
	return string(out)
}
