package templates.gcp.GCPNetworkEnableFirewallLogsConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "compute.googleapis.com/Firewall"

    log_config := lib.get_default(asset.resource.data, "logConfig", {})
    is_enabled := lib.get_default(log_config, "enable", false)
    is_enabled == false

    message := sprintf("Firewall logs are disabled in firewall %v.", [asset.name])
    metadata := {"resource": asset.name}
}