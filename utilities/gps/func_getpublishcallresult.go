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
	"log"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/pubsub"
)

// GetPublishCallResult func to be used in go routine to scale pubsub event publish
func GetPublishCallResult(ctx context.Context, publishResult *pubsub.PublishResult, waitgroup *sync.WaitGroup, msgInfo string, pubSubErrNumber *uint64, pubSubMsgNumber *uint64, logEventEveryXPubSubMsg uint64) {
	defer waitgroup.Done()
	id, err := publishResult.Get(ctx)
	if err != nil {
		log.Printf("ERROR count %d on %s: %v", atomic.AddUint64(pubSubErrNumber, 1), msgInfo, err)
		return
	}
	msgNumber := atomic.AddUint64(pubSubMsgNumber, 1)
	if msgNumber%logEventEveryXPubSubMsg == 0 {
		// No retry on pubsub publish as already implemented in the GO client
		log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
	}
	// log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
}
