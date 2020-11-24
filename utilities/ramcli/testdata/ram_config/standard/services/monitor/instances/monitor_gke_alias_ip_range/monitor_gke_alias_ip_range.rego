#Check is cluster has alias ip range
package templates.gcp.GCPGKEEnableAliasIPRangesConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    cluster := asset.resource.data
    alias_ip_ranges_disabled(cluster)

    message := sprintf("Alias IP ranges are disabled in cluster %v.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
alias_ip_ranges_disabled(cluster) {
    ipAllocationPolicy := lib.get_default(cluster, "ipAllocationPolicy", {})
    useIpAliases := lib.get_default(ipAllocationPolicy, "useIpAliases", false)
    useIpAliases != true
}