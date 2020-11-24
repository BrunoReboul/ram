#Check is cluster is using CoreOs or Not
package templates.gcp.GCPGKEContainerOptimizedOSConstraintV1

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
    non_cos_image(node_pool)

    message := sprintf("Cluster %v has node pool %v without Container-Optimized OS.", [asset.name, node_pool.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
non_cos_image(node_pool) {
    nodeConfig := lib.get_default(node_pool, "config", {})
    imageType := lib.get_default(nodeConfig, "imageType", "")
    imageType != "COS"
}