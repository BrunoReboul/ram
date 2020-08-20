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

package validater

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestValidater(t *testing.T) {
	type isNotZeroValueString struct {
		S string `valid:"isNotZeroValue"`
	}
	type isNotZeroValueInt64 struct {
		I int64 `valid:"isNotZeroValue"`
	}
	type isNotZeroValueSlice struct {
		Sl []string `valid:"isNotZeroValue"`
	}
	type isAvailableMemoryint64 struct {
		A int64 `valid:"isAvailableMemory"`
	}
	type levelC struct {
		IsNotZeroValueString isNotZeroValueString
		IsNotZeroValueInt64  isNotZeroValueInt64
	}
	type levelB struct {
		IsAvailableMemoryint64 isAvailableMemoryint64
		LevelC                 levelC
	}
	type levelA struct {
		IsNotZeroValueSlice isNotZeroValueSlice
		LevelB              levelB
	}
	var tests = []struct {
		name                 string
		structure            interface{}
		pedigree             string
		wantValidation       bool
		wantErrorMsgCount    int
		wantErrorMsgContains []string
	}{
		{
			name:           "IsNotZeroValueStringProvided",
			structure:      isNotZeroValueString{"BlaBla"},
			pedigree:       "my/pe/di/gree",
			wantValidation: true,
		},
		{
			name:              "IsNotZeroValueStringEmpty",
			structure:         isNotZeroValueString{""},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 1,
			wantErrorMsgContains: []string{
				"my/pe/di/gree",
			},
		},
		{
			name:           "IsNotZeroValueInt64Provided",
			structure:      isNotZeroValueInt64{123},
			pedigree:       "my/pe/di/gree",
			wantValidation: true,
		},
		{
			name:              "IsNotZeroValueInt64Empty",
			structure:         isNotZeroValueInt64{0},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 1,
			wantErrorMsgContains: []string{
				"my/pe/di/gree",
			},
		},
		{
			name:           "IsNotZeroValueSliceProvided",
			structure:      isNotZeroValueSlice{[]string{"a", "b"}},
			pedigree:       "my/pe/di/gree",
			wantValidation: true,
		},
		{
			name:              "IsNotZeroValueSliceEmpty",
			structure:         isNotZeroValueSlice{[]string{}},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 1,
			wantErrorMsgContains: []string{
				"my/pe/di/gree",
			},
		},
		{
			name:           "IsAvailableMemoryint64Valid",
			structure:      isAvailableMemoryint64{128},
			pedigree:       "my/pe/di/gree",
			wantValidation: true,
		},
		{
			name:              "IsAvailableMemoryint64Invalid",
			structure:         isAvailableMemoryint64{99},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 1,
			wantErrorMsgContains: []string{
				"my/pe/di/gree",
			},
		},
		{
			name: "levelCOneInvalid",
			structure: levelC{
				IsNotZeroValueString: isNotZeroValueString{"BlaBla"},
				IsNotZeroValueInt64:  isNotZeroValueInt64{0},
			},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 1,
			wantErrorMsgContains: []string{
				"my/pe/di/gree/IsNotZeroValueInt64",
			},
		},
		{
			name: "levelBThreeInvalidOnThree",
			structure: levelB{
				IsAvailableMemoryint64: isAvailableMemoryint64{12},
				LevelC: levelC{
					IsNotZeroValueString: isNotZeroValueString{""},
					IsNotZeroValueInt64:  isNotZeroValueInt64{0},
				},
			},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 3,
			wantErrorMsgContains: []string{
				"my/pe/di/gree/IsAvailableMemoryint64",
				"my/pe/di/gree/LevelC/IsNotZeroValueString",
				"my/pe/di/gree/LevelC/IsNotZeroValueInt64",
			},
		},
		{
			name: "levelATwoInvalidOnFour",
			structure: levelA{
				IsNotZeroValueSlice: isNotZeroValueSlice{[]string{}},
				LevelB: levelB{
					IsAvailableMemoryint64: isAvailableMemoryint64{128},
					LevelC: levelC{
						IsNotZeroValueString: isNotZeroValueString{"BlaBla"},
						IsNotZeroValueInt64:  isNotZeroValueInt64{0},
					},
				},
			},
			pedigree:          "my/pe/di/gree",
			wantValidation:    false,
			wantErrorMsgCount: 2,
			wantErrorMsgContains: []string{
				"my/pe/di/gree/IsNotZeroValueSlice",
				"my/pe/di/gree/LevelB/LevelC/IsNotZeroValueInt64",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer
			log.SetOutput(&buffer)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			err := ValidateStruct(test.structure, test.pedigree)
			errorMsgString := buffer.String()
			// t.Log(countRune(errorMsgString, '\n'))
			// t.Log("Error message list:" + string('\n') + errorMsgString)

			foundErrorMsgCount := countRune(errorMsgString, '\n')
			if test.wantErrorMsgCount != foundErrorMsgCount {
				t.Errorf("Want %d error messages, got %d", test.wantErrorMsgCount, foundErrorMsgCount)
				t.Log("Error message list:" + string('\n') + errorMsgString)
			}

			if len(test.wantErrorMsgContains) > 0 {
				for _, expectedString := range test.wantErrorMsgContains {
					if !strings.Contains(errorMsgString, expectedString) {
						t.Errorf("Error message should contains '%s' and is", expectedString)
						t.Log(string('\n') + errorMsgString)
					}
				}
			}

			if test.wantValidation {
				if err != nil {
					t.Errorf("Want NO error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Should send back an error and is NOT")
				}
			}
		})
	}
}

func countRune(s string, r rune) int {
	count := 0
	for _, c := range s {
		if c == r {
			count++
		}
	}
	return count
}
