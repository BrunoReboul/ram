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

// Package ram avoid code redundancy by grouping types and functions used by other ram packages
package ram

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"
)

// GetByteSet return a set of lenght contiguous bytes starting at bytes
func GetByteSet(start byte, length int) []byte {
	byteSet := make([]byte, length)
	for i := range byteSet {
		byteSet[i] = start + byte(i)
	}
	return byteSet
}

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

// IntialRetryCheck performs intitial controls
// 1) return true and metadata when controls are passed
// 2) return false when controls failed:
// - 2a) with an error to retry the cloud function entry point function
// - 2b) with nil to stop the cloud function entry point function
func IntialRetryCheck(ctxEvent context.Context, initFailed bool, retryTimeOutSeconds int64) (bool, *metadata.Metadata, error) {
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return false, metadata, fmt.Errorf("metadata.FromContext: %v", err) // RETRY
	}
	if initFailed {
		log.Println("ERROR - init function failed")
		return false, metadata, nil // NO RETRY
	}

	// Ignore events that are too old.
	expiration := metadata.Timestamp.Add(time.Duration(retryTimeOutSeconds) * time.Second)
	if time.Now().After(expiration) {
		log.Printf("ERROR - too many retries for expired event '%q'", metadata.EventID)
		return false, metadata, nil // NO MORE RETRY
	}
	return true, metadata, nil
}

// PrintEnptyInterfaceType discover the type below an empty interface
func PrintEnptyInterfaceType(value interface{}, valueName string) error {
	switch t := value.(type) {
	default:
		log.Printf("type %T for value named: %s\n", t, valueName)
	}
	return nil
}
