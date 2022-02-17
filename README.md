# injector

The `injector` is a utility that retrieves a GCP Secret Manager document which contains an HJSON or JSON object with an
embedded top-level `environment` property under which environment variable values are defined. These values are injected
into a shell environment in which a target command is run. The end result is that you can decouple the storage location
and perhaps maintenance of such values from other workflows (e.g. container deployment).

Secret Manager documents are encrypted at rest and these values are pulled at runtime instead of being baked into a
`Docker` image, for example, which provides extra levels of security. Furthermore, the values will not appear in a
process table at runtime.

## Human JSON (HJSON) format

The format of Secret Manager secrets _should be_ [Human JSON (HJSON)](https://hjson.github.io/) or standard JSON. It is
ighly recommended that the HJSON format is used, however, because it supports comments (which means you can document the
file).

For more information on Human JSON refer to the following links:

* https://github.com/hjson/hjson-go
* https://github.com/hjson/vscode-hjson

>NOTE: There is a VSCode plugin for Human JSON (HJSON).

## Usage

The `inject` command implements a number of options (detailed below):

```bash
NAME:
   inject - Handle signals and inject environment variables from GCP secret manager.

USAGE:
   inject [global options] command [command options] [arguments...]

VERSION:
   v1.0.0-beta14

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --key-file value, -k value        Path to file containing JSON format service account key.
   --key-value value, -K value       Base64 encoded string containing JSON format service account key. [$INJECTOR_KEY_VALUE]
   --format-shell, -e                Parse secret contents and convert to exported shell key/value settings.
   --format-shell-unexported, -u     Parse secret contents and convert to unexported shell key/value settings.
   --format-json, -j                 Parse secret contents and convert from hJSON to JSON.
   --format-raw, -r                  Output unparsed secret contents. This will likely be hJSON or JSON.
   --ignore, -i                      Ignore missing secret options.
   --ignore-preserve-env, -I         Ignore missing secret options, pass environment variables from parent OS into command shell.
   --preserve-env, -E                Pass environment variables from parent OS into command shell.
   --output-file value, -o value     Write output to file. Default is stdout; passing "-" also represents stdout.
   --project value, -p value         GCP project id. [$INJECTOR_PROJECT]
   --secret-name value, -S value     Name of secret containing environment variables and values. [$INJECTOR_SECRET_NAME]
   --secret-version value, -V value  Version of secret containing environment variables and values. ("latest" if not specified) [$INJECTOR_SECRET_VERSION]
   --debug, -d                       Show debug information.
   --help, -h                        show help
   --version, -v                     print the version
```

## Typical usage from the command line

### Retrieve the latest version of a secret and output as JSON

The following command will retrieve the latest version of the secret specified by name from the Secret Manager in the
project specified by id.

```bash
prompt> inject --key-value <KEY_VALUE> --project <PROJECT_ID> --secret-name "<SECRET_NAME>" --format-json
```

Take note that the `--format-json` option indicates that a __Human JSON__ document will be converted to JSON.

### Wrapping a command

To invoke (wrap) a command so that it has access to retrieved environment variables simply specify a command name as the
final argument:

```bash
prompt> inject --key-value <KEY_VALUE> --project <PROJECT_ID> --secret-name "<SECRET_NAME>" <COMMAND>
```

## Required options

A few of the `inject` options must be defined to retrieve a Secret Manager document:

* --key-value, -K (a JSON format GCP service account that grants access to the specific secret)
* --project, -p (the GCP project id)
* --secret-name, -S (the name of Secret Manager secret)

The above options can be defined on the command line as options or as environment variables that are available
from inside of the shell where the `inject` command will be run. The recognized environment variable names for
these options are:

* INJECTOR_KEY_VALUE
* INJECTOR_PROJECT
* INJECTOR_SECRET_NAME

As an option, the `inject` command also supports the specification of a secret version which can be defined on
the command line:

- --secret-version, -V (the secret version to retrieve)

If no secret version is specified then `inject` will assume `latest` (i.e. the most-recent secret version will be
retrieved.

### Priority of Options specified multiple ways

If an `inject` option has been defined as both an environment variable and a command line flag, the flag will take
priority.

## Wrapping PID1 for Docker containers

Since the goal of the injector is to retrieve and _inject_ environment variables into the environment so that these
values are available at runtime the underlying runtime needs to be wrapped. This means that the injector will call the
target program. An

To support a wrapper A `Dockerfile` might include the following:

```bash
ARG INJECTOR_REL='1.0.0-beta14'
...

# Install injector
RUN set -x \
    && cd "${BUILD_TEMP}" \
    && curl -sSL "https://github.com/markeissler/injector/releases/download/v${INJECTOR_REL}/linux_amd64.tar.gz" -o "injector-${INJECTOR_REL}.tar.gz" \
    && tar xvzf "injector-${INJECTOR_REL}.tar.gz" \
    && cp "inject" "/usr/local/bin/inject-${INJECTOR_REL}" \
    && chown -R root:root "/usr/local/bin/inject-${INJECTOR_REL}" \
    && chmod 0755 "/usr/local/bin/inject-${INJECTOR_REL}" \
    && ln -s "/usr/local/bin/inject-${INJECTOR_REL}" "/usr/local/bin/inject"
...

# default command (startup supervisor)
CMD ["/usr/local/bin/inject", "--ignore-preserve-env", "/path/to/target/app"]
```

## Format of the secret document

The injector will parse the secret document so that environment variable names are generated from a parent up to its
last child property in a descending tree. Consider the following document:

```HJSON
{
    // Environment variables are specified as key/value pairs where the key is specified here in lowercase snake case
    // but will be converted to uppercase snake case prior to injection into the container at boot.
    "environment": {
        "app": {
            "debug": "0"
        },
        "buckets": {
            "backups": "my-backups-bucket",
            "storage": "my-storage-bucket"
        },
        // Unless you explicitly pass through the PATH environment variable (e.g. using the `-E` option) injector will
        // discard the PATH all together! You can inject a new (possibly more restrictive) PATH by specifying one here.
        "path": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    }
}
```

The injector will generate the following environment variables from the above document:

* APP_DEBUG="0"
* BUCKETS_BACKUPS="my-backups-bucket"
* BUCKETS_STORAGE="my-storage-bucket"
* PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

Notice that the top-level `environment` property is MANDATORY but will be pruned.

> NOTE: As indicated in the comments for the example document, the `path` must be defined unless you choose to inherit
> the path from the parent environment using the `--preserve-env, -E` option.

## Preserving environment variables from the parent OS

Most of the time you will not want to provide an isolated environment to the wrapped command, possibly to prevent
undesirable leakage. If you need to pass through environment variables from the parent environment you can specify the
`--preserve-env, -E` option. Beware that the injector will overwrite the values of any inherited environment variables
with those that have similar names in the retrieved secret.

---
