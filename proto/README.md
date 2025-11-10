## Protobuf generation

Default generation (messages + clients):
- Runs via `npm run generate` (or `bun generate`) in `proto/`

### PHP notes
- The official grpc-php generator produces client stubs (e.g. `PluginClient`), not server stubs.
- If you want to implement a PHP gRPC server, use OpenSwoole:
  - Install the OpenSwoole extension
    - macOS: `pecl install openswoole` (or `brew install openswoole` if available)
    - Linux (Debian/Ubuntu): `sudo pecl install openswoole` (or `sudo apt install php-openswoole` if available)
  - Install the compiler for server stubs
    - Vendor (recommended): `composer require --dev openswoole/grpc-compiler` (use `vendor/bin/protoc-gen-openswoole-grpc`)
    - Or system-wide: install `protoc-gen-openswoole-grpc` and ensure it’s on PATH (e.g., via package manager or manual binary)
  - Then either replace the PHP generator in `buf.gen.php.yaml` to use the OpenSwoole plugin or run `protoc` directly with `--openswoole-grpc_out`
- We keep the repo’s default PHP generation as grpc-php (clients only) to avoid extra toolchain requirements. Developers can opt into OpenSwoole locally if needed.


