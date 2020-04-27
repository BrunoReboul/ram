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

package gsr

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/sourcerepo/v1"
)

// Deploy RepoDeployment check if the repo exist, if not try to create it
func (repoDeployment *RepoDeployment) Deploy() (err error) {
	log.Printf("%s gsr source repositories", repoDeployment.Core.InstanceName)
	projectsService := repoDeployment.Core.Services.SourcerepoService.Projects
	projectName := fmt.Sprintf("projects/%s", repoDeployment.Core.SolutionSettings.Hosting.ProjectID)
	repoName := fmt.Sprintf("%s/repos/%s", projectName, repoDeployment.Core.SolutionSettings.Hosting.Repository.Name)
	repo, err := projectsService.Repos.Get(repoName).Context(repoDeployment.Core.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), "notFound") {
			var repoToCreate sourcerepo.Repo
			repoToCreate.Name = repoName
			repo, err = projectsService.Repos.Create(projectName, &repoToCreate).Context(repoDeployment.Core.Ctx).Do()
			if err != nil {
				return err
			}
			log.Printf("%s gsr source repo created %s", repoDeployment.Core.InstanceName, repo.Name)
		} else {
			return err
		}
	}
	log.Printf("%s gsr found source repo %s", repoDeployment.Core.InstanceName, repo.Name)
	return nil
}
