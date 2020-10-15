#Check is cluster is using StackDriver Logging
package templates.gcp.GCPGKEEnableStackdriverLoggingConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"

    cluster := asset.resource.data
    stackdriver_logging_disabled(cluster)

    message := sprintf("Stackdriver logging is disabled in cluster %v.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
stackdriver_logging_disabled(cluster) {
    loggingService := lib.get_default(cluster, "loggingService", "none")
    loggingService != ""
    loggingService != "logging.googleapis.com"
    loggingService != "logging.googleapis.com/kubernetes"
}