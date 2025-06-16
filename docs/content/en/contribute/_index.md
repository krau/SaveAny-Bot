---
title: "Contributing"
weight: 20
---

# Contributing

## Contributing New Storage Backend

1. Fork this repository and clone it to your local machine.
2. Add the new storage backend type in `pkg/enums/storage/storages.go` and run code generation.
3. Define the storage backend configuration in the `config/storage` directory and add it to `config/storage/factory.go`.
4. Create a new package in the `storage` directory, implement the storage backend, and import it in `storage/storage.go`.
5. Update the documentation to include configuration details for the new storage backend.