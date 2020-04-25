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
	"bytes"
	"io/ioutil"
	"net/http"
)

// UploadZipUsingSignedURL upload the rile content using a signed URL
func (functionDeployment *FunctionDeployment) UploadZipUsingSignedURL() (response *http.Response, err error) {
	contentBytes, err := ioutil.ReadFile(functionDeployment.Artifacts.CloudFunctionZipFullPath)
	if err != nil {
		return response, err
	}
	request, err := http.NewRequest("PUT", functionDeployment.Artifacts.CloudFunction.SourceUploadUrl,
		bytes.NewReader(contentBytes))
	if err != nil {
		return response, err
	}
	request.Header.Add("content-type", "application/zip")
	request.Header.Add("x-goog-content-length-range", "0,104857600")
	httpClient := new(http.Client)
	response, err = httpClient.Do(request)
	if err != nil {
		return response, err
	}
	return response, nil
}
