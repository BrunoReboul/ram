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

package templates.gcp.GCPSQLAllowedAuthorizedNetworksConstraintV1

import data.validator.gcp.lib as lib

deny[{
	"msg": message,
	"details": metadata,
}] {
	constraint := input.constraint
	lib.get_constraint_params(constraint, params)
	# If blackList mode then 0 expected matches, else 1
	mode := lib.get_default(params, "mode", "blacklist")
	target_match_count(mode, desired_count)

	asset := input.asset
	asset.asset_type == "sqladmin.googleapis.com/Instance"
	asset.resource.data.ipAddresses[_].type != "PRIVATE"
	matched_networks := seek_networks(params, asset.resource.data.settings.ipConfiguration)


	# # Assess compliance
	# # Blacklist (0) and found (1): 0 <> 1 = TRUE, so deny rule is true, it reports msg and details
	# # Blacklist (0) and NOT found (0): 0 <> 0 = FALSE, so deny rule is false, no outputs
	# # Whitelist (1) and found (1): 1 <> 1 = FALSE, so deny rule is false, no outputs
	# # Whitelist (1) and NOT found (0): 1 <> 0 = TRUE, so deny rule is true, it reports msg and details
	count(matched_networks) != desired_count
	
	message := sprintf("%v has authorized networks that are not allowed: %v", [asset.name, matched_networks])
	metadata := {"resource": asset.name, "network":matched_networks}
}

seek_networks(params, ipConfiguration) = result {
	constraint_networks = lib.get_default(params, "networks", ["0.0.0.0/0"])
	configured_networks := {network |
		network = ipConfiguration.authorizedNetworks[_].value
	}

	matched_networks := {network |
		network = configured_networks[_]
		constraint_networks[_] == network
	}

	result := matched_networks
}

# deny[{
# 	"msg": message,
# 	"details": metadata,
# }] {
# 	constraint := input.constraint
# 	lib.get_constraint_params(constraint, params)

# 	asset := input.asset
# 	asset.asset_type == "sqladmin.googleapis.com/Instance"

# 	check_ssl(params, asset.resource.data.settings.ipConfiguration) == false

# 	message := sprintf("%v has networks with SSL settings in violation of policy", [asset.name])
# 	metadata := {"resource": asset.name}
# }
# check_ssl(params, ipConfiguration) = result {
# 	lib.has_field(params, "ssl_enabled") == false
# 	result = true
# }

# check_ssl(params, ipConfiguration) = result {
# 	requireSsl := lib.get_default(ipConfiguration, "requireSsl", false)
# 	result = requireSsl == params.ssl_enabled
# }

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
