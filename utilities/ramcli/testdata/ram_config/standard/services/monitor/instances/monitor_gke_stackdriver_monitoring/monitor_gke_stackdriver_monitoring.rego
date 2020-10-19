#Check is cluster is using CoreOs or Not
package templates.gcp.GCPGKEEnableStackdriverMonitoringConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    cluster := asset.resource.data
    stackdriver_monitoring_disabled(cluster)

    message := sprintf("Stackdriver monitoring is disabled in cluster %v.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
stackdriver_monitoring_disabled(cluster) {
    monitoringService := lib.get_default(cluster, "monitoringService", "none")
    monitoringService != "monitoring.googleapis.com"
}