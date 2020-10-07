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

package str

import (
	"bytes"
	"fmt"
	"sort"
)

// FlattenMapStringString flatten a string string map into a string key1="value1", keyx="valuex",
func FlattenMapStringString(m map[string]string) string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	b := new(bytes.Buffer)
	for _, k := range keys {
		fmt.Println(k, m[k])
		fmt.Fprintf(b, "%s=\"%s\", ", k, m[k])
	}
	return b.String()
}
