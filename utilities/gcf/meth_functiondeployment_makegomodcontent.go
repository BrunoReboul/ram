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
	"time"
)

// goMod go.mod skeleton, replace first %s goVersion, second by ramVersion
const goMod = `
// generated code %v

module example.com/cloudfunction

go %s

require github.com/BrunoReboul/ram %s
`

// MakeGoModContent craft the content of a cloud function go.mod file for a RAM microservice instance
func (functionDeployment *FunctionDeployment) makeGoModContent() (goModContent string) {
	return fmt.Sprintf(goMod, time.Now(), functionDeployment.Core.GoVersion, functionDeployment.Core.RAMVersion)
}
