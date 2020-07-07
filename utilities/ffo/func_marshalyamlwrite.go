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

package ffo

import (
	"io/ioutil"

	"github.com/BrunoReboul/ram/utilities/str"
	"gopkg.in/yaml.v2"
)

// MarshalYAMLWrite Marshal an interface to YAML format and write the bytes to a given path
func MarshalYAMLWrite(path string, v interface{}) (err error) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, append([]byte(str.YAMLDisclaimer), bytes...), 0644)
	if err != nil {
		return err
	}
	return nil
}
