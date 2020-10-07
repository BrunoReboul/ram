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
	"fmt"
	"testing"
)

func TestUnitIsNotTransientElseWait(t *testing.T) {
	var testCases = []struct {
		name                   string
		errMessage             string
		shouldNotFindTransient bool
	}{
		{
			name:                   "err403",
			errMessage:             "403 forbidden",
			shouldNotFindTransient: true,
		},
		{
			name:                   "err500",
			errMessage:             "500 Internal Server Error",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err501",
			errMessage:             "501 Not Implemented",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err502",
			errMessage:             "502 Bad Gateway",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err503",
			errMessage:             "503 Service Unavailable",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err504",
			errMessage:             "504 Gateway Timeout",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err505",
			errMessage:             "505 HTTP Version Not Supported",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err506",
			errMessage:             "506 Variant Also Negotiates",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err507",
			errMessage:             "507 Insufficient Storage",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err508",
			errMessage:             "508 Loop Detected",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err510",
			errMessage:             "510 Not Extended",
			shouldNotFindTransient: false,
		},
		{
			name:                   "err511",
			errMessage:             "511 Network Authentication Required",
			shouldNotFindTransient: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := fmt.Errorf(tc.errMessage)
			result := IsNotTransientElseWait(err, 0)
			if tc.shouldNotFindTransient {
				if tc.shouldNotFindTransient != result {
					t.Errorf("Should NOT find transient in errMessage %s", tc.errMessage)
				}
			} else {
				if tc.shouldNotFindTransient != result {
					t.Errorf("Should find transient in errMessage %s", tc.errMessage)
				}
			}
		})
	}
}
