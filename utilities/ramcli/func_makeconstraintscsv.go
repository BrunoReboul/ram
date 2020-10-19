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

package ramcli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/solution"
)

func makeConstraintsCSV(repositoryPath string, constraintFolderRelativePaths []string) (records [][]string, err error) {
	type constraintCSV struct {
		Kind     string
		Metadata struct {
			Name        string
			Annotations struct {
				Description string
				Category    string
			}
		}
		Spec struct {
			Severity string
		}
	}
	records = [][]string{
		{
			"serviceName", "ruleName", "constraintName", "severity", "category", "kind", "description",
		},
	}
	for _, constraintFolderRelativePath := range constraintFolderRelativePaths {
		// fmt.Println(constraintFolderRelativePath)
		parts := strings.Split(constraintFolderRelativePath, "/")
		microserviceName := parts[3]
		parts = strings.Split(microserviceName, "_")
		pathServiceName := parts[1]
		pathRuleName := strings.Replace(microserviceName, fmt.Sprintf("%s_%s_", parts[0], parts[1]), "", 1)

		var constraint constraintCSV
		err = ffo.ReadUnmarshalYAML(fmt.Sprintf("%s/%s/constraint.yaml", repositoryPath, constraintFolderRelativePath), &constraint)
		if err != nil {
			return records, err
		}

		record := []string{pathServiceName,
			pathRuleName,
			constraint.Metadata.Name,
			constraint.Spec.Severity,
			constraint.Metadata.Annotations.Category,
			constraint.Kind,
			constraint.Metadata.Annotations.Description}
		records = append(records, record)
	}
	file, err := os.OpenFile(fmt.Sprintf("%s/%s/monitor/constraints.csv", repositoryPath, solution.MicroserviceParentFolderName),
		os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		return records, err
	}
	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(records)
	csvWriter.Flush()
	return records, nil
}
