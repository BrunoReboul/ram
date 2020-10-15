 package templates.gcp.GCPSQLSSLConstraintV1

       import data.validator.gcp.lib as lib

       deny[{
        "msg": message,
        "details": metadata,
       }] {
        asset := input.asset
        asset.asset_type == "sqladmin.googleapis.com/Instance"

        settings := asset.resource.data.settings

        ipConfiguration := lib.get_default(settings, "ipConfiguration", {})
        requireSsl := lib.get_default(ipConfiguration, "requireSsl", false)
        requireSsl == false

        message := sprintf("%v does not require SSL", [asset.name])
        metadata := {"resource": asset.name}
       }