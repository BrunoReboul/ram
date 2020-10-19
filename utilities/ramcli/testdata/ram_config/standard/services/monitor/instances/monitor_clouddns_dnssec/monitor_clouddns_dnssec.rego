package templates.gcp.GCPDNSSECConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "dns.googleapis.com/ManagedZone"

    asset.resource.data.dnssecConfig.state != "ON"

    message := sprintf("%v: DNSSEC is not enabled.", [asset.name])
    metadata := {"resource": asset.name}
}