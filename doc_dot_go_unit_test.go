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

package string

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const docDotGoName = "doc.go"

var levelOneFolders = []string{"services", "utilities"}

func TestUnitDocDotGo(t *testing.T) {
	for _, levelOneFolder := range levelOneFolders {
		err := filepath.Walk("./"+levelOneFolder, func(path string, info os.FileInfo, err error) error {
			levelOneFolder := levelOneFolder // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				if err != nil {
					t.Fatal(err)
				}
				if info.IsDir() && !strings.HasSuffix(path, levelOneFolder) && !strings.Contains(path, "testdata") {
					_, err = os.Stat(path + "/" + docDotGoName)
					if os.IsNotExist(err) {
						t.Errorf("%v: missing %s file in this subfolder", path, docDotGoName)
					}
				}
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
