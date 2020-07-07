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

	"gopkg.in/yaml.v2"
)

// ReadUnmarshalYAML Read bytes from a given path and unmarshal assuming YAML format
func ReadUnmarshalYAML(path string, settings interface{}) (err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, settings)
	if err != nil {
		return err
	}
	return nil
}
