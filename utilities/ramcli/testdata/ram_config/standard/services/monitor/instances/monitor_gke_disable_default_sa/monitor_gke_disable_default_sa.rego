package templates.gcp.GCPGKEDisableDefaultServiceAccountConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    cluster := asset.resource.data
    node_pools := lib.get_default(cluster, "nodePools", [])
    node_pool := node_pools[_]
    default_service_account(node_pool)

    message := sprintf("Cluster %v has node pool %v with default service account.", [asset.name, node_pool.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
default_service_account(node_pool) {
    nodeConfig := lib.get_default(node_pool, "config", {})
    serviceAccount := lib.get_default(nodeConfig, "serviceAccount", "default")
    serviceAccount == "default"
}