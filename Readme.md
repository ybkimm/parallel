# Parallel

Parallel is a command line utility for executing multiple commands in parallel.

# Usage

```
$ go run go.ybk.im/parallel "echo First" "echo Second"
[parallel#0] Process /usr/bin/echo (#1) is running
[echo#1]     A
[parallel#0] Process /usr/bin/echo (#2) is running
[echo#2]     B
[parallel#0] All processes were closed
```
