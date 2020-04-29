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

package cai

import "strings"

// GetAssetShortTypeName returns a short version of asset type <serviceName>-<assetType>, like bigquery-Dataset. It deals with k8s exceptions
func GetAssetShortTypeName(assetType string) string {
	var serviceName string
	tmpArr := strings.Split(assetType, "/")
	assetTypeName := tmpArr[len(tmpArr)-1]
	serviceTypeName := tmpArr[0]
	switch serviceTypeName {
	case "rbac.authorization.k8s.io":
		serviceName = "k8srbac"
	case "extensions.k8s.io":
		serviceName = "k8sextensions"
	case "networking.k8s.io":
		serviceName = "k8snetworking"
	default:
		tmpArr := strings.Split(assetType, ".")
		serviceName = tmpArr[0]
	}
	assetShortTypeName := serviceName + "-" + assetTypeName
	return assetShortTypeName
}
