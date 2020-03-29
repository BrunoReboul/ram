# RAM Real-time Asset Monitor

## What

Audit Google Cloud resources (the assets) compliance against a set of rules when the resource is updated. The stream of detected non compliances could then be consumed to alert, report or even fix on the fly.

### Use cases

1. Security compliance, usually 80% of the rules
2. Operational compliance
   - E.g. each Cloud SQL MySQL instance should have a defined maintenance window to avoid downtime
3. Financial Operations (finOps) compliance
   - E.g. Do not provision anymore N1 virtual machines instances, instead provision N2: the price performance ratio is better

## Why

- It is all easier to fix when it is detected early
- Value is delivered only when a detected non compliance is fixed
