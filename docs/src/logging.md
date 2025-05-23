# Logging
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

CloudNativePG outputs logs in JSON format directly to standard output, including
PostgreSQL logs, without persisting them to storage for security reasons. This
design facilitates seamless integration with most Kubernetes-compatible log
management tools, including command line ones like
[stern](https://github.com/stern/stern).

!!! Important
    Long-term storage and management of logs are outside the scope of the
    operator and should be handled at the Kubernetes infrastructure level.
    For more information, see the
    [Kubernetes Logging Architecture](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
    documentation.

Each log entry includes the following fields:

- `level` – The log level (e.g., `info`, `notice`).
- `ts` – The timestamp.
- `logger` – The type of log (e.g., `postgres`, `pg_controldata`).
- `msg` – The log message, or the keyword `record` if the message is in JSON
  format.
- `record` – The actual record, with a structure that varies depending on the
  `logger` type.
- `logging_pod` – The name of the pod where the log was generated.

!!! Info
    If your log ingestion system requires custom field names, you can rename
    the `level` and `ts` fields using the `log-field-level` and
    `log-field-timestamp` flags in the operator controller. This can be configured
    by editing the `Deployment` definition of the `cloudnative-pg` operator.

## Cluster Logs

You can configure the log level for the instance pods in the cluster
specification using the `logLevel` option. Available log levels are: `error`,
`warning`, `info` (default), `debug`, and `trace`.

!!! Important
    Currently, the log level can only be set at the time the instance starts.
    Changes to the log level in the cluster specification after the cluster has
    started will only apply to new pods, not existing ones.

## Operator Logs

The logs produced by the operator pod can be configured with log
levels, same as instance pods: `error`, `warning`, `info` (default), `debug`,
and `trace`.

The log level for the operator can be configured by editing the `Deployment`
definition of the operator and setting the `--log-level` command line argument
to the desired value.

## PostgreSQL Logs

Each PostgreSQL log entry is a JSON object with the `logger` key set to
`postgres`. The structure of the log entries is as follows:

```json
{
  "level": "info",
  "ts": 1619781249.7188137,
  "logger": "postgres",
  "msg": "record",
  "record": {
    "log_time": "2021-04-30 11:14:09.718 UTC",
    "user_name": "",
    "database_name": "",
    "process_id": "25",
    "connection_from": "",
    "session_id": "608be681.19",
    "session_line_num": "1",
    "command_tag": "",
    "session_start_time": "2021-04-30 11:14:09 UTC",
    "virtual_transaction_id": "",
    "transaction_id": "0",
    "error_severity": "LOG",
    "sql_state_code": "00000",
    "message": "database system was interrupted; last known up at 2021-04-30 11:14:07 UTC",
    "detail": "",
    "hint": "",
    "internal_query": "",
    "internal_query_pos": "",
    "context": "",
    "query": "",
    "query_pos": "",
    "location": "",
    "application_name": "",
    "backend_type": "startup"
  },
  "logging_pod": "cluster-example-1",
}
```

!!! Info
    Internally, the operator uses PostgreSQL's CSV log format. For more details,
    refer to the [PostgreSQL documentation on CSV log format](https://www.postgresql.org/docs/current/runtime-config-logging.html).

## PGAudit Logs

CloudNativePG offers seamless and native support for
[PGAudit](https://www.pgaudit.org/) on PostgreSQL clusters.

To enable PGAudit, add the necessary `pgaudit` parameters in the `postgresql`
section of the cluster configuration.

!!! Important
    The PGAudit library must be added to `shared_preload_libraries`.
    CloudNativePG automatically manages this based on the presence of `pgaudit.*`
    parameters in the PostgreSQL configuration. The operator handles both the
    addition and removal of the library from `shared_preload_libraries`.

Additionally, the operator manages the creation and removal of the PGAudit
extension across all databases within the cluster.

!!! Important
    CloudNativePG executes the `CREATE EXTENSION` and `DROP EXTENSION` commands
    in all databases within the cluster that accept connections.

The following example demonstrates a PostgreSQL `Cluster` deployment with
PGAudit enabled and configured:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-example
spec:
  instances: 3

  postgresql:
    parameters:
      "pgaudit.log": "all, -misc"
      "pgaudit.log_catalog": "off"
      "pgaudit.log_parameter": "on"
      "pgaudit.log_relation": "on"

  storage:
    size: 1Gi
```

The audit CSV log entries generated by PGAudit are parsed and routed to
standard output in JSON format, similar to all other logs:

- `.logger` is set to `pgaudit`.
- `.msg` is set to `record`.
- `.record` contains the entire parsed record as a JSON object. This structure
  resembles that of `logging_collector` logs, with the exception of
  `.record.audit`, which contains the PGAudit CSV message formatted as a JSON
  object.

This example shows sample log entries:

```json
{
  "level": "info",
  "ts": 1627394507.8814096,
  "logger": "pgaudit",
  "msg": "record",
  "record": {
    "log_time": "2021-07-27 14:01:47.881 UTC",
    "user_name": "postgres",
    "database_name": "postgres",
    "process_id": "203",
    "connection_from": "[local]",
    "session_id": "610011cb.cb",
    "session_line_num": "1",
    "command_tag": "SELECT",
    "session_start_time": "2021-07-27 14:01:47 UTC",
    "virtual_transaction_id": "3/336",
    "transaction_id": "0",
    "error_severity": "LOG",
    "sql_state_code": "00000",
    "backend_type": "client backend",
    "audit": {
      "audit_type": "SESSION",
      "statement_id": "1",
      "substatement_id": "1",
      "class": "READ",
      "command": "SELECT FOR KEY SHARE",
      "statement": "SELECT pg_current_wal_lsn()",
      "parameter": "<none>"
    }
  },
  "logging_pod": "cluster-example-1",
}
```

See the
[PGAudit documentation](https://github.com/pgaudit/pgaudit/blob/master/README.md#format) <!-- wokeignore:rule=master -->
for more details about each field in a record.

## Other Logs

All logs generated by the operator and its instances are in JSON format, with
the `logger` field indicating the process that produced them. The possible
`logger` values are as follows:

- `barman-cloud-wal-archive`: logs from `barman-cloud-wal-archive`
- `barman-cloud-wal-restore`: logs from `barman-cloud-wal-restore`
- `initdb`: logs from running `initdb`
- `pg_basebackup`: logs from running `pg_basebackup`
- `pg_controldata`: logs from running `pg_controldata`
- `pg_ctl`: logs from running any `pg_ctl` subcommand
- `pg_rewind`: logs from running `pg_rewind`
- `pgaudit`: logs from the PGAudit extension
- `postgres`: logs from the `postgres` instance (with `msg` distinct from
  `record`)
- `wal-archive`: logs from the `wal-archive` subcommand of the instance manager
- `wal-restore`: logs from the `wal-restore` subcommand of the instance manager
- `instance-manager`: from the [PostgreSQL instance manager](./instance_manager.md)

With the exception of `postgres`, which follows a specific structure, all other
`logger` values contain the `msg` field with the escaped message that is
logged.
