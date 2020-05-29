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

package ram

import (
	"log"
	"os"
)

// CheckPath crashes execution when the path is not found or when an other error occurs
func CheckPath(path string) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Path does not exist: %s", path)
		} else {
			log.Fatalln(err)
		}
	}
}
