# lacune

## Install

```sh
go install github.com/alesr/lacune@latest
```

## Features

- visualize inline code coverage
- re-run tests on demand
- scroll through and search code
- measure GC behavior of your test suite (wall time + gctrace telemetry)

![lacune demo](assets/demo.gif)

## Usage

Run `lacune` in the root directory of your Go project (where your `go.mod` file is located). It will automatically detect your test files and coverage profiles.

## GC report

Press **`b`** in the TUI to open the GC report overlay. Lacune runs **your project's test suite** (`go test ./...`) under whatever garbage collector is the default for your Go toolchain, then shows how your code behaves under that GC.

| Go version (from `go.mod`) | Default GC   |
| -------------------------- | ------------ |
| 1.25 and earlier           | Classic GC   |
| 1.26+                      | Green Tea GC |

The report measures:

- **Wall time** — how long the full test run took
- **GC cycles** — number of GC events during the run
- **Total GC pause** — sum of stop-the-world pause time
- **GC CPU** — share of CPU spent in GC
- **Peak heap** — largest heap size observed during a GC cycle

Telemetry comes from `GODEBUG=gctrace=1`. A compile-only warm-up pass runs first so wall time reflects test execution, not compilation.

### Keys

| Key       | Action                                              |
| --------- | --------------------------------------------------- |
| `b`       | Open / close GC report                              |
| `+` / `-` | More / fewer iterations (`go test -count`, default **3**) |
| `t`       | Toggle Docker isolation (pinned 2 CPU cores, 512 MB RAM) |
| `Enter`   | Run                                                 |
| `?`       | Metric guide (in results view)                      |
| `Esc`     | Cancel (while running) or close                     |

This measures your **test suite**, not production traffic — but it exercises *your* code, not a synthetic benchmark. Docker and host runs use different environments, so compare like with like.

The logic lives in [`pkg/gcbench`](pkg/gcbench/) and is wired into the TUI from `main.go`.

## Contribute

Contributions are welcome! Open an issue or submit a PR.
