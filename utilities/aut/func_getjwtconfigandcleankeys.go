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
	"context"
	"encoding/base64"

	"golang.org/x/oauth2/jwt"
)

// getJWTConfigAndCleanKeys build a JWT config and manage the init state
func getJWTConfigAndCleanKeys(ctx context.Context,
	serviceAccountEmail string,
	keyJSONFilePath string,
	projectID string,
	gciAdminUserToImpersonate string,
	scopes []string,
	serviceAccountKeyNames []string,
	initID string,
	microserviceName string,
	instanceName string,
	environment string) (
	jwtConfig *jwt.Config,
	err error) {
	keyRestAPIFormat, err := getKeyJSONdataAndCleanKeys(ctx,
		serviceAccountEmail,
		keyJSONFilePath,
		projectID,
		serviceAccountKeyNames,
		initID,
		microserviceName,
		instanceName,
		environment)
	if err != nil {
		return jwtConfig, err
	}

	// Convert format
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-go
	keyJSONdata, err := base64.StdEncoding.DecodeString(keyRestAPIFormat.PrivateKeyData)
	if err != nil {
		return jwtConfig, err
	}

	//DEBUG
	// var keyConsoleFormat keyConsoleFormat
	// err = json.Unmarshal(keyJSONdata, &keyConsoleFormat)
	// if err != nil {
	// 	return jwtConfig, err
	// }
	// JSONMarshalIndentPrint(keyConsoleFormat)

	// using Json Web joken a the method with cerdentials does not yet implement the subject impersonification
	// https://github.com/googleapis/google-api-java-client/issues/1007
	jwtConfig, err = getJWTConfigAndImpersonate(keyJSONdata, gciAdminUserToImpersonate, scopes)
	if err != nil {
		return jwtConfig, err
	}
	return jwtConfig, nil
}
