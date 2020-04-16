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
	"log"
	"os"
)

// GetChild returns the list to relative subfolders path for a folder + a relative path, crashes execution on errors
func GetChild(basePath string, relativeFolderPath string) []string {
	var childRelativePaths []string
	folderPath := fmt.Sprintf("%s/%s", basePath, relativeFolderPath)
	CheckPath(folderPath)
	folder, err := os.Open(folderPath)
	if err != nil {
		log.Fatalln(err)
	}
	folderInfo, err := folder.Readdir(-1)
	folder.Close()
	if err != nil {
		log.Fatalln(err)
	}
	for _, child := range folderInfo {
		if child.IsDir() {
			childRelativePath := fmt.Sprintf("%s/%s", relativeFolderPath, child.Name())
			childRelativePaths = append(childRelativePaths, childRelativePath)
		}
	}
	return childRelativePaths
}
