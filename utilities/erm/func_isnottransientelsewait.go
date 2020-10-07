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

package erm

import (
	"log"
	"strings"
	"time"
)

// IsNotTransientElseWait check is the error is a 5xx and wait if it is
func IsNotTransientElseWait(err error, waitSec time.Duration) (isNotTransient bool) {
	erroMessage := err.Error()
	transientErrors := []string{"500", "501", "502", "503", "504", "505", "506", "507", "508", "510", "511"}
	isNotTransient = true
	for _, transientError := range transientErrors {
		if strings.Contains(erroMessage, transientError) {
			isNotTransient = false
			break
		}
	}
	if !isNotTransient {
		log.Printf("Transient error, wait %d sec and retry %s", waitSec, erroMessage)
		time.Sleep(waitSec * time.Second)
	}
	return isNotTransient
}
