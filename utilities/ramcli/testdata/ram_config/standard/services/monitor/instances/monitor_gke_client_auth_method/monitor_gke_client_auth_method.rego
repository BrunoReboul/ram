#Check is client auth method
package templates.gcp.GCPGKERestrictClientAuthenticationMethodsConstraintV1

import data.validator.gcp.lib as lib

deny[{
    "msg": message,
    "details": metadata,
}] {
    constraint := input.constraint
    asset := input.asset
    asset.asset_type == "container.googleapis.com/Cluster"
    cluster := asset.resource.data
    master_auth := lib.get_default(cluster, "masterAuth", {})

    not check_all_disabled(master_auth)

    message := sprintf("%v has client certificate or static password authentication enabled.", [asset.name])
    metadata := {"resource": asset.name}
}

###########################
# Rule Utilities
###########################
check_all_disabled(master_auth) {
    # For clusters before v1.12, if masterAuth is unspecified, username will
    # be set to "admin", a random password will be generated, and a client certificate
    # will be issued.
    master_auth != {}
    auth_with_client_cert_disabled(master_auth) == true
    auth_with_static_password_disabled(master_auth) == true
}

auth_with_client_cert_disabled(master_auth) {
    # Scan for clientCertificateConfig to make sure issueClientCertificate is false.
    client_cert_config := lib.get_default(master_auth, "clientCertificateConfig", {})
    client_cert_enabled := lib.get_default(client_cert_config, "issueClientCertificate", false)
    client_cert_enabled == false
}

auth_with_static_password_disabled(master_auth) {
    # Scan for masterAuth to make sure it’s specified and username to
    # make sure it’s empty or unspecified.
    user_name := lib.get_default(master_auth, "username", "")
    user_name == ""
}