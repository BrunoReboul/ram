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

package gps

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/pubsub"
	"github.com/BrunoReboul/ram/utilities/glo"
)

// GetPublishCallResult func to be used in go routine to scale pubsub event publish
func GetPublishCallResult(ctx context.Context,
	publishResult *pubsub.PublishResult,
	waitgroup *sync.WaitGroup,
	msgInfo string,
	pubSubErrNumber *uint64,
	pubSubMsgNumber *uint64,
	logEventEveryXPubSubMsg uint64,
	pubSubID string,
	microserviceName string,
	instanceName string,
	environment string) {
	defer waitgroup.Done()
	id, err := publishResult.Get(ctx)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName:   microserviceName,
			InstanceName:       instanceName,
			Environment:        environment,
			Severity:           "WARNING",
			Message:            "publishResult.Get(ctx)",
			Description:        fmt.Sprintf("count %d on %s: %v", atomic.AddUint64(pubSubErrNumber, 1), msgInfo, err),
			TriggeringPubsubID: pubSubID,
		})
		return
	}
	msgNumber := atomic.AddUint64(pubSubMsgNumber, 1)

	// debug log
	// log.Println(glo.Entry{
	// 	MicroserviceName:   microserviceName,
	// 	InstanceName:       instanceName,
	// 	Environment:        environment,
	// 	Severity:           "INFO",
	// 	Message:            fmt.Sprintf("GetPublishCallResult %s pubSubMsgNumber %d", msgInfo, msgNumber),
	// 	Description:        fmt.Sprintf("id %s", id),
	// 	TriggeringPubsubID: pubSubID,
	// })
	// end debug log

	if msgNumber%logEventEveryXPubSubMsg == 0 {
		// No retry on pubsub publish as already implemented in the GO client
		log.Println(glo.Entry{
			MicroserviceName:   microserviceName,
			InstanceName:       instanceName,
			Environment:        environment,
			Severity:           "INFO",
			Message:            fmt.Sprintf("progression %d messages published", msgNumber),
			Description:        fmt.Sprintf("now %s id %s", msgInfo, id),
			TriggeringPubsubID: pubSubID,
		})
	}
}
