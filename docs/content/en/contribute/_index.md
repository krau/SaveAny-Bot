---
title: "Contributing"
weight: 20
---

# Contributing

Before you start, please fork this repository, clone it locally, and set up your Go development environment.

Here are some guidelines and suggestions for contributing code. You don't have to strictly follow them, but they help speed up review and merging:

- **Open an issue before adding new features**, so we can discuss design and implementation details and avoid work that doesn't fit the project design.
- **Use modern development tools**, format your code before committing, and keep the style consistent.
- **Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)**, and avoid vague or overly simple commit messages.

## Contributing New Storage Backend

1. Add the new storage backend type in `pkg/enums/storage/storages.go` and run code generation.
2. Define the storage backend configuration in the `config/storage` directory and add it to `config/storage/factory.go`.
3. Create a new package in the `storage` directory, implement the storage backend, and import it in `storage/storage.go`.
4. Update the documentation to include configuration details for the new storage backend.

## Contributing New Parsers

You can either implement native parsers in Go (recommended), or write JavaScript-based parser plugins.

If you use Go:

1. Create a new package under the `parsers` directory and implement the parser logic.
2. Register the parser in the `init` function in `parsers/parser.go`.

If you use JavaScript:

1. Refer to `plugins/example_parser_basic.js` as an example.
2. Create a new `.js` file in the `plugins` directory and implement your parsing logic there.

Note: Parsers under the `plugins` directory are not embedded into the binary by default. Users must download them manually and place them in the configured plugin directories to enable them.