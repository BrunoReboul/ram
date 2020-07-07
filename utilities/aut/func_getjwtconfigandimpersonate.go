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

package aut

import (
	"log"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

// getJWTConfigAndImpersonate build JWT with impersonification
func getJWTConfigAndImpersonate(keyJSONdata []byte, gciAdminUserToImpersonate string, scopes []string) (jwtConfig *jwt.Config, err error) {
	// using Json Web joken a the method with cerdentials does not yet implement the subject impersonification
	// https://github.com/googleapis/google-api-java-client/issues/1007

	// scope constants: https://godoc.org/google.golang.org/api/admin/directory/v1#pkg-constants
	jwtConfig, err = google.JWTConfigFromJSON(keyJSONdata, scopes...)
	if err != nil {
		log.Printf("google.JWTConfigFromJSON: %v", err)
		return jwtConfig, err
	}
	jwtConfig.Subject = gciAdminUserToImpersonate

	// DEBUG
	// jwtConfigJSON, err := json.Marshal(jwtConfig)
	// log.Printf("jwt %s", string(jwtConfigJSON))

	return jwtConfig, nil
}
