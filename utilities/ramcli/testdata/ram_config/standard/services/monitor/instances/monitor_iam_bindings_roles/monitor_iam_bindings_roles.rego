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

package templates.gcp.GCPIAMBindingsrolesConstraintV1

import data.validator.gcp.lib as lib

deny[{
	"msg": message,
	"details": metadata,
}] {
	constraint := input.constraint
	lib.get_constraint_params(constraint, params)

	# If blackList mode then 0 expected matches, else 1
	mode := lib.get_default(params, "mode", "whitelist")
	target_match_count(mode, desired_count)

	# Get asset metadata
	asset := input.asset
	# For each binding
	binding := asset.iam_policy.bindings[_]

	# Rule to filter on asset type
	# If find this asset type in the constraint asset type list then match count = 1, 1 eq 1 = TRUE
	# Else not found and this rule is FALSE, so deny parent rule is false too 
	asset_type := asset.asset_type
	asset_types := lib.get_default(params, "assettypes", {"**"})
	matches_type := [t | t = asset_types[_]; glob.match(t, [], asset_type)]
	count(matches_type) == 1

	# if asset binding role is one of the constraint list role then count = 1 else 0
	role := binding.role
	matches_found = [r | r = params.roles[_]; glob.match(r, [], role)]

	# Assess compliance
	# Backlist (0) and found (1): 0 <> 1 = TRUE, so deny rule is true, it reports msg and details
	# Backlist (0) and NOT found (0): 0 <> 0 = FALSE, so deny rule is false, no outputs
	# whitelist (1) and found (1): 1 <> 1 = FALSE, so deny rule is false, no outputs
	# whitelist (1) and NOT found (0): 1 <> 0 = TURE, so deny rule is true, it reports msg and details
	count(matches_found) != desired_count

	message := sprintf("IAM policy for %v uses role %v, which should not be", [asset.name, role])

	metadata := {
		"resource": asset.name,
		"role": role,
	}
}

###########################
# Rule Utilities
###########################

# Determine the overlap between matches under test and constraint
target_match_count(mode) = 0 {
	mode == "blacklist"
}

target_match_count(mode) = 1 {
	mode == "whitelist"
}
