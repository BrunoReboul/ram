package templates.gcp.GCPGKELegacyAbacConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    container := asset.resource.data
    enabled := legacy_abac_enabled(container)
    enabled == true

    message := sprintf("%v has legacy ABAC enabled.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
legacy_abac_enabled(container) = legacy_abac_enabled {
    legacy_abac := lib.get_default(container, "legacyAbac", {})
    legacy_abac_enabled := lib.get_default(legacy_abac, "enabled", false)
}