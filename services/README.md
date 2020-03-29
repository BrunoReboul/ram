# Service packages structure

All service cloud function packages share a consistent structure

## Two functions and one type

### `Initialize` function

- Goal
  - Optimize cloud function performance by reducing the invocation latency
- Implementation
  - Is executed once per cloud function instance as a cold start.
  - Cache objects expensive to create, like clients
  - Retreive settings once, like environment variables
  - Cached objects and reteived settings are exposed in one global variable named `global`
  
### `Global` type

- A `struct` to define a global variable carrying cached objects and retreived settings by `Initialized` function and used by `EntryPoint` function

### `EntryPoint` function

- Goal
  - Execute operations to be performed each time the cloud function is invoked
- Implementation
  - Is executed on every event triggering the cloud function
  - Uses cached objects and retreived settings prepared by the `Initialized` function and carried by a global variable of type `Global`
  - Performs the task a given service is targetted to do that is described before the `package` key word

## Implementation example

How to implement a RAM service package in a Google Cloud function?

### `go.mod`

Replace `<vX.Y.Z>` in the go.mod file by the RAM version [release](https://github.com/BrunoReboul/ram/releases) to be used

```go
module example.com/cloudfunction

go 1.11

require github.com/BrunoReboul/ram <vX.Y.Z>
```

### `function.go` for a background function triggered by PubSub events

Replace `<package_name>` by the name of the service package to be used

```go
// Package p contains a background cloud function
package p

import (
    "context"

    "github.com/BrunoReboul/ram/<package_name>"
    "github.com/BrunoReboul/ram/ram"
)

var global <package_name>.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage) error {
    return <package_name>.EntryPoint(ctxEvent, PubSubMessage, &global)
}

func init() {
    <package_name>.Initialize(ctx, &global)
}
```

### `function.go` for a background function triggered by Google Cloud Storage events

Replace `<package_name>` by the name of the service package to be used

```go
// Package p contains a background cloud function
package p

import (
    "context"

    "github.com/BrunoReboul/ram/<package_name>"
    "github.com/BrunoReboul/ram/ram"
)

var global <package_name>.Global
var ctx = context.Background()

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, gcsEvent ram.GCSEvent) error {
    return <package_name>.EntryPoint(ctxEvent, PubSubMessage, &global)
}

func init() {
    <package_name>.Initialize(ctx, &global)
}
```

## Automatic retrying

Automatic retrying is consistently implemented in RAM service packages as documented in Google Cloud Function best practice [Retrying Background Functions](https://cloud.google.com/functions/docs/bestpractices/retries)

Impact on cloud functions Stackdriver logs:

- Errors entry in log only appears for transient errors.
  - As en error is reported the function is retried.
- Other errors and loged as information to avoid unwanted retries
  - To find errors in such cloud function logs you can use the following Stackdriver logging filter

```txt
resource.type="cloud_function"
textPayload:"error"
```
