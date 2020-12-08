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

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
)

// GetClientOptionAndCleanKeys build a clientOption object and manage the init state
func GetClientOptionAndCleanKeys(ctx context.Context,
	serviceAccountEmail string,
	keyJSONFilePath string,
	projectID string,
	gciAdminUserToImpersonate string,
	scopes []string,
	serviceAccountKeyNames []string,
	logEntryPrefix string) (
	option.ClientOption, bool) {
	var clientOption option.ClientOption
	var jwtConfig *jwt.Config

	jwtConfig, err := getJWTConfigAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, projectID, gciAdminUserToImpersonate, scopes, serviceAccountKeyNames, logEntryPrefix)
	if err != nil {
		return clientOption, false
	}

	httpClient := jwtConfig.Client(ctx)
	// Use client option as admin.New(httpClient) is deprecated https://godoc.org/google.golang.org/api/admin/directory/v1#New
	clientOption = option.WithHTTPClient(httpClient)

	return clientOption, true
}
