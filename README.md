# injector

The `injector` is a utility that retrieves a GCP Secret Manager document which contains an HJSON or JSON object with an
embedded top-level `environment` property under which environment variable values are defined. These values are injected
into a shell environment in which a target command is run. The end result is that you can decouple the storage location
and perhaps maintenance of such values from other workflows (e.g. container deployment).

Secret Manager documents are encrypted at rest and these values are pulled at runtime instead of being baked into a
`Docker` image which provides extra levels of security. Furthermore, the values will not appear in a process table at
runtime.

## Usage

__TBD__

---
AlphaFlow Inc.
