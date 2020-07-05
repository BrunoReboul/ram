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
	"fmt"
	"io/ioutil"
	"log"
)

// ExploreFolder list folder childs and their type
func ExploreFolder(path string) (err error) {
	filesInfo, err := ioutil.ReadDir("path")
	if err != nil {
		return fmt.Errorf("ioutil.ReadDir %v", err)
	}
	log.Printf("List child in %s", path)
	for _, fileInfo := range filesInfo {
		log.Printf("Parent %s base name %v IsDir %v Size (bytes) %d modified %v",
			path,
			fileInfo.IsDir(),
			fileInfo.Name(),
			fileInfo.Size(), fileInfo.ModTime())
	}
	return nil
}
