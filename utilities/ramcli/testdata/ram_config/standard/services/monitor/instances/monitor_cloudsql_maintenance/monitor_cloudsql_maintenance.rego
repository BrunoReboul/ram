package templates.gcp.GCPSQLMaintenanceWindowConstraintV1

import data.validator.gcp.lib as lib

# A violation is generated only when the rule body evaluates to true.
deny[{
    "msg": message,
    "details": metadata,
}] {
    # by default any hour accepted
    default_hours := {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
    spec := lib.get_default(input.constraint, "spec", "")
    parameters := lib.get_default(spec, "parameters", "")
    maintenance_window_hours := lib.get_default(parameters, "hours", default_hours)
    desired_maintenance_window_hours := get_default_when_empty(maintenance_window_hours, default_hours)
    exempt_list := lib.get_default(parameters, "exemptions", [])

    # Verify that resource is Cloud SQL instance and is not a first gen
    # Maintenance window is supported only on 2nd generation instances
    asset := input.asset
    asset.asset_type == "sqladmin.googleapis.com/Instance"
    asset.resource.data.backendType != "FIRST_GEN"

    # Check if resource is in exempt list
    matches := {asset.name} & cast_set(exempt_list)
    count(matches) == 0

    # get instance settings
    settings := lib.get_default(asset.resource.data, "settings", {})
    instance_maintenance_window := lib.get_default(settings, "maintenanceWindow", {})
    instance_maintenance_window_hour := lib.get_default(instance_maintenance_window, "hour", "")

    # check compliance
    hour_matches := {instance_maintenance_window_hour} & cast_set(desired_maintenance_window_hours)
    count(hour_matches) == 0

    message := sprintf("%v missing or incorrect maintenance window. Hour: '%v'", [asset.name, instance_maintenance_window_hour])
    metadata := {"resource": asset.name}
}

# Rule utilities
get_default_when_empty(value, default_value) = output {
    count(value) != 0
    output := value
}

get_default_when_empty(value, default_value) = output {
    count(value) == 0
    output := default_value
}