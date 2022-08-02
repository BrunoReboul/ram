# Compliance rules summary

Repository: **standard**

*Timestamp* 2022-08-01 14:29:21.003028668 +0200 CEST m=+0.018431138

Service | rules | constraints
--- | --- | ---
**clouddns** | 2 | 2
**cloudsql** | 4 | 5
**gae** | 1 | 1
**gce** | 3 | 3
**gke** | 11 | 11
**iam** | 3 | 5
**kms** | 1 | 1

7 services 25 rules 28 constraints

## clouddns

- dnssec   - **[clouddns_dnssec](instances/monitor_clouddns_dnssec/constraints/clouddns_dnssec/readme.md)** (*major* )
- rsasha1   - **[clouddns_rsasha1](instances/monitor_clouddns_rsasha1/constraints/clouddns_rsasha1/readme.md)** (*critical* )

## cloudsql

- backup   - **[cloudsql_backup](instances/monitor_cloudsql_backup/constraints/cloudsql_backup/readme.md)** (*major* )
- maintenance   - **[cloudsql_maintenance](instances/monitor_cloudsql_maintenance/constraints/cloudsql_maintenance/readme.md)** (*low* )
- networkacl
  - **[no_public_access](instances/monitor_cloudsql_networkacl/constraints/no_public_access/readme.md)** (*major* )
  - **[no_zscaler_access](instances/monitor_cloudsql_networkacl/constraints/no_zscaler_access/readme.md)** (*medium* )
- ssl   - **[cloudsql_ssl](instances/monitor_cloudsql_ssl/constraints/cloudsql_ssl/readme.md)** (*major* )

## gae

- max_version   - **[traffic_split_to_more_than_2_versions](instances/monitor_gae_max_version/constraints/traffic_split_to_more_than_2_versions/readme.md)** (*medium* )

## gce

- external_ip   - **[gce_external_ip](instances/monitor_gce_external_ip/constraints/gce_external_ip/readme.md)** (*major* )
- network_firewall_log   - **[gce_network_firewall_log](instances/monitor_gce_network_firewall_log/constraints/gce_network_firewall_log/readme.md)** (*medium* )
- network_firewall_rules   - **[no_public_access](instances/monitor_gce_network_firewall_rules/constraints/no_public_access/readme.md)** (*critical* )

## gke

- abac   - **[gke_abac](instances/monitor_gke_abac/constraints/gke_abac/readme.md)** (*low* )
- alias_ip_range   - **[gke_alias_ip_range](instances/monitor_gke_alias_ip_range/constraints/gke_alias_ip_range/readme.md)** (*medium* )
- allow_node_sa   - **[gke_allow_node_sa](instances/monitor_gke_allow_node_sa/constraints/gke_allow_node_sa/readme.md)** (*medium* )
- client_auth_method   - **[gke_client_auth_method](instances/monitor_gke_client_auth_method/constraints/gke_client_auth_method/readme.md)** (*major* )
- coreos   - **[gke_coreos](instances/monitor_gke_coreos/constraints/gke_coreos/readme.md)** (*medium* )
- dashboard   - **[gke_dashboard](instances/monitor_gke_dashboard/constraints/gke_dashboard/readme.md)** (*major* )
- disable_default_sa   - **[gke_disable_default_sa](instances/monitor_gke_disable_default_sa/constraints/gke_disable_default_sa/readme.md)** (*medium* )
- private_cluster   - **[gke_private_cluster](instances/monitor_gke_private_cluster/constraints/gke_private_cluster/readme.md)** (*major* )
- stackdriver_logging   - **[gke_stackdriver_logging](instances/monitor_gke_stackdriver_logging/constraints/gke_stackdriver_logging/readme.md)** (*major* )
- stackdriver_monitoring   - **[gke_stackdriver_monitoring](instances/monitor_gke_stackdriver_monitoring/constraints/gke_stackdriver_monitoring/readme.md)** (*major* )
- version   - **[gke_version](instances/monitor_gke_version/constraints/gke_version/readme.md)** (*low* )

## iam

- bindings_roles
  - **[no_billing_account_creator](instances/monitor_iam_bindings_roles/constraints/no_billing_account_creator/readme.md)** (*low* )
  - **[no_primitives_on_org_and_folders](instances/monitor_iam_bindings_roles/constraints/no_primitives_on_org_and_folders/readme.md)** (*major* )
- members
  - **[no_domains_grants](instances/monitor_iam_members/constraints/no_domains_grants/readme.md)** (*major* )
  - **[no_public_access](instances/monitor_iam_members/constraints/no_public_access/readme.md)** (*major* )
- sa_key_age   - **[iam_sa_key_age](instances/monitor_iam_sa_key_age/constraints/iam_sa_key_age/readme.md)** (*medium* )

## kms

- rotation   - **[rotation_100_days_max](instances/monitor_kms_rotation/constraints/rotation_100_days_max/readme.md)** (*medium* )
