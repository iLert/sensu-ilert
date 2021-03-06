# Sensu iLert Handler

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
  - [Help output](#help-output)
  - [Deduplication key](#deduplication-key)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Environment variables](#environment-variables)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu iLert Handler is a [Sensu Event Handler][3] which manages
[iLert][2] incidents, for alerting operators. With this handler,
[Sensu][1] can trigger and resolve iLert incidents.

## Usage examples

### Help output
```
The Sensu Go Ilert handler for incident management

Usage:
  sensu-ilert-handler [flags]
  sensu-ilert-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -k, --dedup-key-template string   The Ilert deduplication key template, can be set with ILERT_DEDUP_KEY_TEMPLATE (default "{{.Entity.Name}}-{{.Check.Name}}")
  -d, --details-template string     The template for the alert details, can be set with ILERT_DETAILS_TEMPLATE (default full event JSON)
  -h, --help                        help for sensu-ilert-handler
  -S, --summary-template string     The template for the alert summary, can be set with ILERT_SUMMARY_TEMPLATE (default "{{.Entity.Name}}/{{.Check.Name}} : {{.Check.Output}}")
  -t, --token string                The Ilert API authentication token, can be set with ILERT_SENSU_TOKEN

Use "sensu-ilert-handler [command] --help" for more information about a command.
```

### Deduplication key

The deduplication key is determined via the `--dedup-key-template` argument.  It
is a Golang template containing the event values and defaults to
`{{.Entity.Name}}-{{.Check.Name}}`.


## Configuration
### Asset registration

The easiest way to get this handler added to your Sensu environment, is to add it as an asset from Bonsai:

```sh
sensuctl asset add ilert/sensu-ilert --rename sensu-ilert-handler
```

See `sensuctl asset --help` for details on how to specify version.

### Handler definition

```yml
type: Handler
api_version: core/v2
metadata:
  name: ilert
  namespace: default
spec:
  type: pipe
  command: sensu-ilert-handler
  timeout: 10
  runtime_assets:
  - ilert/sensu-ilert
  filters:
  - is_incident
  secrets:
  - name: ILERT_SENSU_TOKEN
    secret: ilert_sensu_token
```

### Environment variables

Most arguments for this handler are available to be set via environment
variables.  However, any arguments specified directly on the command line
override the corresponding environment variable.

|Argument            |Environment Variable        |
|--------------------|----------------------------|
|--token             |ILERT_SENSU_TOKEN           |
|--summary-template  |ILERT_SUMMARY_TEMPLATE      |
|--dedup-key-template|ILERT_DEDUP_KEY_TEMPLATE    |

**Security Note:** Care should be taken to not expose the auth token for this
handler by specifying it on the command line or by directly setting the
environment variable in the handler definition.  It is suggested to make use of
[secrets management][4] to surface it as an environment variable.  The handler
definition above references it as a secret.  Below is an example secrets
definition that make use of the built-in [env secrets provider][5].

```yml
---
type: Secret
api_version: secrets/v1
metadata:
  name: ilert_sensu_token
spec:
  provider: env
  id: ILERT_SENSU_TOKEN
```

## Installation from source

Download the latest version of the sensu-ilert-handler from [releases][6],
or create an executable from this source.

From the local path of the sensu-ilert-handler repository:
```
go build -o /usr/local/bin/sensu-ilert-handler
```

## Contributing

See https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md

[1]: https://github.com/sensu/sensu-go
[2]: https://www.ilert.com/
[3]: https://docs.sensu.io/sensu-go/5.0/reference/handlers/#how-do-sensu-handlers-workdynamic-notifications#section-eventalert-severity-levels
[4]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/
[5]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/
[6]: https://github.com/iLert/sensu-ilert