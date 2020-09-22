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

package ramcli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// getVersions look for a go.mod in the curent path, returns Go and RAM versions, crashes execution on errors
func getVersions(repositoryPath string) (goVersion, ramVersion string, err error) {
	goModFilePath := repositoryPath + "/go.mod"
	if _, err := os.Stat(goModFilePath); err != nil {
		return "", "", err
	}
	file, err := os.Open(goModFilePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "go ") {
			goVersion = strings.TrimLeft(line, "go ")
		}
		if !(strings.Contains(line, "module") || strings.Contains(line, "replace")) {
			if strings.Contains(line, "github.com/") && strings.Contains(line, "/ram ") {
				parts := strings.Split(line, "/ram ")
				ramVersion = parts[len(parts)-1]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if goVersion == "" {
		return "", "", fmt.Errorf("goVersion NOT found, missing go x.y line in go.mod")
	}
	if ramVersion == "" {
		return "", "", fmt.Errorf("ramVersion NOT found, missing required line to ram module in go.mod")
	}
	return goVersion, ramVersion, nil
}
