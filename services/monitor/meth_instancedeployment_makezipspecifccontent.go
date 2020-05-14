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

package monitor

import (
	"fmt"
	"io/ioutil"

	"github.com/BrunoReboul/ram/utilities/ram"
)

// audit.rego code
const auditRego = `
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

package validator.gcp.lib

audit[result] {
	asset := data.assets[_]

	constraints := data.constraints
	constraint := constraints[_]
	spec := _get_default(constraint, "spec", {})
	match := _get_default(spec, "match", {})
	# Default matcher behavior is to match everything.
	target := _get_default(match, "target", ["organization/*"])
	gcp := _get_default(match, "gcp", {})
	gcp_target := _get_default(gcp, "target", target)
	re_match(gcp_target[_], asset.ancestry_path)
	exclude := _get_default(match, "exclude", [])
	gcp_exclude := _get_default(gcp, "exclude", exclude)
	exclusion_match := {asset.ancestry_path | re_match(gcp_exclude[_], asset.ancestry_path)}
	count(exclusion_match) == 0

	violations := data.templates.gcp[constraint.kind].deny with input.asset as asset
		 with input.constraint as constraint

	violation := violations[_]

	result := {
		"asset": asset.name,
		"constraint": constraint.metadata.name,
		"constraint_config": constraint,
		"violation": violation,
	}
}

# has_field returns whether an object has a field
_has_field(object, field) {
	object[field]
}

# False is a tricky special case, as false responses would create an undefined document unless
# they are explicitly tested for
_has_field(object, field) {
	object[field] == false
}

_has_field(object, field) = false {
	not object[field]
	not object[field] == false
}

# get_default returns the value of an object's field or the provided default value.
# It avoids creating an undefined state when trying to access an object attribute that does
# not exist
_get_default(object, field, _default) = output {
	_has_field(object, field)
	output = object[field]
}

_get_default(object, field, _default) = output {
	_has_field(object, field) == false
	output = _default
}
`

// constraints.rego code
const constraintsRego = `
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

package validator.gcp.lib

# Function to fetch the constraint spec
# Usage:
# get_constraint_params(constraint, params)

get_constraint_params(constraint) = params {
	params := constraint.spec.parameters
}

# Function to fetch constraint info
# Usage:
# get_constraint_info(constraint, info)

get_constraint_info(constraint) = info {
	info := {
		"name": constraint.metadata.name,
		"kind": constraint.kind,
	}
}
`

// util.rego code
const utilRego = `
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

package validator.gcp.lib

# has_field returns whether an object has a field
has_field(object, field) {
	object[field]
}

# False is a tricky special case, as false responses would create an undefined document unless
# they are explicitly tested for
has_field(object, field) {
	object[field] == false
}

has_field(object, field) = false {
	not object[field]
	not object[field] == false
}

# get_default returns the value of an object's field or the provided default value.
# It avoids creating an undefined state when trying to access an object attribute that does
# not exist
get_default(object, field, _default) = output {
	has_field(object, field)
	output = object[field]
}

get_default(object, field, _default) = output {
	has_field(object, field) == false
	output = _default
}
`

// MakeZipSpecificContent
func (instanceDeployment *InstanceDeployment) makeZipSpecificContent() (specificZipFiles map[string]string, err error) {
	specificZipFiles = make(map[string]string)
	specificZipFiles["opa/modules/audit.rego"] = auditRego
	specificZipFiles["opa/modules/constraints.rego"] = constraintsRego
	specificZipFiles["opa/modules/util.rego"] = utilRego

	regoRuleFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s.rego",
		instanceDeployment.Core.RepositoryPath,
		ram.MicroserviceParentFolderName,
		instanceDeployment.Core.ServiceName,
		ram.InstancesFolderName,
		instanceDeployment.Core.InstanceName,
		instanceDeployment.Core.InstanceName)
	bytes, err := ioutil.ReadFile(regoRuleFilePath)
	if err != nil {
		return make(map[string]string), err
	}
	specificZipFiles[fmt.Sprintf("opa/modules/%s.rego", instanceDeployment.Core.InstanceName)] = string(bytes)

	regoConstraintsFolderPath := fmt.Sprintf("%s/%s/%s/%s/%s/%s.rego",
		instanceDeployment.Core.RepositoryPath,
		ram.MicroserviceParentFolderName,
		instanceDeployment.Core.ServiceName,
		ram.InstancesFolderName,
		instanceDeployment.Core.InstanceName,
		ram.RegoConstraintsFolderName)
	childs, err := ioutil.ReadDir(regoConstraintsFolderPath)
	if err != nil {
		return make(map[string]string), err
	}
	for _, child := range childs {
		if child.IsDir() {
			constraintName := child.Name()
			bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/constraint.yaml", regoConstraintsFolderPath, constraintName))
			if err != nil {
				return make(map[string]string), err
			}
			specificZipFiles[fmt.Sprintf("opa/constraints/%s/constraint.yaml", constraintName)] = string(bytes)
		}
	}
	return specificZipFiles, err
}
