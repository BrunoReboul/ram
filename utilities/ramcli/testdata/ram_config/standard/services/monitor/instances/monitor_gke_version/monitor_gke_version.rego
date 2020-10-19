package templates.gcp.GCPGKEVersionConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    lib.get_constraint_params(constraint, params)

    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    container := asset.resource.data

    minimal_master_version := lib.get_default(params, "master_versions", "1.14.10")
    minimal_node_version := lib.get_default(params, "node_versions", "1.14.10")

    check(params, container.currentMasterVersion, minimal_master_version, container.currentNodeVersion, minimal_node_version)
    message := sprintf("%v Master and/or Node is not in an allowed version.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################

check(params, currentMasterVersion, minimal_master_version, currentNodeVersion, minimal_node_version) {
    check_version(params, currentMasterVersion, minimal_master_version)
}

check(params, currentMasterVersion, minimal_master_version, currentNodeVersion, minimal_node_version) {
    check_version(params, currentNodeVersion, minimal_node_version)
}

check_version(params, currentVersion, minimal_version) {
    # Get first part of the version "x.x.x"-gke.xx
    current_version := split(currentVersion, "-gke")
    current_version_parts := split(current_version[0], ".")
    minimal_version_parts := split(minimal_version, ".")

    trace(concat("-", current_version_parts))
    trace(concat("-", minimal_version_parts))
    compare_first_digit(current_version_parts, minimal_version_parts)
}

compare_first_digit(current_version_parts, minimal_version_parts) {
    to_number(current_version_parts[0]) < to_number(minimal_version_parts[0])
}

compare_first_digit(current_version_parts, minimal_version_parts) {
    to_number(current_version_parts[0]) == to_number(minimal_version_parts[0])
    compare_second_digit(current_version_parts, minimal_version_parts)
}

compare_second_digit(current_version_parts, minimal_version_parts) {
    to_number(current_version_parts[1]) < to_number(minimal_version_parts[1])
}

compare_second_digit(current_version_parts, minimal_version_parts) {
    to_number(current_version_parts[1]) == to_number(minimal_version_parts[1])
    compare_third_digit(current_version_parts, minimal_version_parts)
}

compare_third_digit(current_version_parts, minimal_version_parts) {
    to_number(current_version_parts[2]) < to_number(minimal_version_parts[2])
}