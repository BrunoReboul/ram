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
	"archive/zip"
	"os"
)

// ZipSource make a zip file from a map where the key is the file name and the value the string file content
func ZipSource(zipFullPath string, zipFiles map[string]string) (err error) {
	zipSourceFile, err := os.Create(zipFullPath)
	if err != nil {
		return err
	}
	defer zipSourceFile.Close()
	zipWriter := zip.NewWriter(zipSourceFile)

	for name, strContent := range zipFiles {
		file, err := zipWriter.Create(name)
		if err != nil {
			return err
		}
		_, err = file.Write([]byte(strContent))
		if err != nil {
			return err
		}
	}

	err = zipWriter.Close()
	if err != nil {
		return err
	}
	return nil
}
