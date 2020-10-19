#Check is cluster is private or not
package templates.gcp.GCPGKEPrivateClusterConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    cluster := asset.resource.data
    private_cluster_config := lib.get_default(cluster, "privateClusterConfig", {})
    private_cluster_config == {}

    message := sprintf("Cluster %v is not private.", [asset.name])
    metadata := {"resource": asset.name}
}