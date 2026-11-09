# Go rules

- When testing HTTP servers, consume response bodies before closing them so
  the client can reuse connections. Merely closing unread bodies can leave
  newly opened connections that delay graceful server shutdown. A cleanup
  deadline can otherwise make a passing request test fail during shutdown.
- When receiving a worker's `error` from a channel in tests, assert it rather
  than discarding it. Recent errcheck versions flag bare receives of errors.
- Optional live process output can use `io.MultiWriter` alongside the existing
  capture buffers. Use separate stdout/stderr destinations unless the writer
  explicitly supports concurrent writes. Test that output is observable before
  process completion and that cancellation terminates child processes.
- Use a golangci-lint build that supports the selected Go toolchain. An older
  linter can panic when asked to analyze standard-library files from a newer Go
  release, even when the project's go.mod declares an older language version.
