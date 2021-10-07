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

package gcf

import (
	"fmt"
)

// getRunTime returns the GCF runtime string for a given Go version. Error is version not supported
func getRunTime(goVersion string) (runTime string, err error) {
	switch goVersion {
	case "1.11":
		return "", fmt.Errorf("GO 1.11 has been deprecated on on 2020-08-05 fron the cloud function runtime, provided version was: %s", goVersion)
	case "1.13":
		return "go113", nil
	case "1.16":
		return "go116", nil
	default:
		return "", fmt.Errorf("Supported GO version is 1.13, provided version was: %s", goVersion)
	}
}
