package templates.gcp.GCPDNSSECPreventRSASHA1ConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    lib.get_constraint_params(constraint, params)

    asset := input.asset
    asset.asset_type == "dns.googleapis.com/ManagedZone"

    dnssecConfig := asset.resource.data.dnssecConfig

    keySpec := dnssecConfig.defaultKeySpecs[_]

    keySpec.algorithm == "RSASHA1"

    check_key_type(params, keySpec)

    message := sprintf("%v: DNSSEC has weak RSASHA1 algorithm enabled", [asset.name])
    metadata := {"resource": asset.name}
}

check_key_type(params, keySpec) {
    lib.has_field(params, "keyType")
    keySpec.keyType == params.keyType
}

check_key_type(params, keySpec) {
    not lib.has_field(params, "keyType")
}