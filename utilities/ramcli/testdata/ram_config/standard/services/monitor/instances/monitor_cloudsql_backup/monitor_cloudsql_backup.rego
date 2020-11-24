package templates.gcp.GCPSQLBackupConstraintV1

import data.validator.gcp.lib as lib

# A violation is generated only when the rule body evaluates to true.
deny[{
    "msg": message,
    "details": metadata,
}] {
    # by default any hour accepted
    spec := lib.get_default(input.constraint, "spec", "")
    parameters := lib.get_default(spec, "parameters", "")
    exempt_list := lib.get_default(parameters, "exemptions", [])

    asset := input.asset
    asset.asset_type == "sqladmin.googleapis.com/Instance"

    # Check if resource is in exempt list
    matches := {asset.name} & cast_set(exempt_list)
    count(matches) == 0

    # get instance settings
    settings := lib.get_default(asset.resource.data, "settings", {})
    instance_backupConfiguration := lib.get_default(settings, "backupConfiguration", {})
    instance_backupConfiguration_enabled := lib.get_default(instance_backupConfiguration, "enabled", "")

    # check compliance
    instance_backupConfiguration_enabled != true

    message := sprintf("%v backup not enabled'", [asset.name])
    metadata := {"resource": asset.name}
}