#Check DASHBOARD IS DISABLED


package templates.gcp.GCPGKEDashboardConstraintV1

import data.validator.gcp.lib as lib

deny[{
	"msg": message,
	"details": metadata,
}] {
	constraint := input.constraint
	asset := input.asset
	asset.asset_type == "container.googleapis.com/Cluster"

	container := asset.resource.data
	disabled := dashboard_disabled(container)
	disabled == false

	message := sprintf("%v has kubernetes dashboard enabled.", [asset.name])
	metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
dashboard_disabled(container) = dashboard_disabled {
	addons_config := lib.get_default(container, "addonsConfig", "default")
	dashboard := lib.get_default(addons_config, "kubernetesDashboard", "default")
	dashboard_disabled := lib.get_default(dashboard, "disabled", false)
}
