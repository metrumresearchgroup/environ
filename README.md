# environ

Package environ provides an easy way to store and manipulate environment state. It is thread and concurrency safe.

## Features

The key features are:

- FromOS, a function to inherit the current state.
- The usual Get/Set/Unset functions.
- Keep/Drop functions that work
- Concurrent safety.
- Functions for reading the set in different forms. (slice, map)
- Functions over the set of values (Len())
- Globbing of keys to keep or drop whole collections of values.

## Use

You can capture your current environment and clean it up as a constraint for future executions:

```go
// Gather OS values for passing down to shells, with modification.
env := environ.FromOS()

// keep some things you may need, including a wildcard to all
// github settings.
env.Keep("PATH", "SHELL", "GITHUB_*", "SSH_*")

// Drop specific keys that might confound a process.
env.Drop("GITHUB_SHA")

// using our command package 
out, err := command.New(command.WithEnv(env.AsSlice())).
    Run("gh", "repo", "clone", "cli/cli")
```
