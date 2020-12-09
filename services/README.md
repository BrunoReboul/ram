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

## Automatic retrying

Automatic retrying is consistently implemented in RAM service packages as documented in Google Cloud Function best practice [Retrying Background Functions](https://cloud.google.com/functions/docs/bestpractices/retries)

Impact on cloud functions Stackdriver logs:

- Errors entry in log only appears for transient errors.
  - As en error is reported the function is retried.
- Other errors and loged as information to avoid unwanted retries
  - To find errors in such cloud function logs you can use the following Stackdriver logging filter

```cloudlogging
resource.type="cloud_function"
textPayload:"REDO_ON_TRANSIENT"
```

The standard exponential backoff algorithm leads to have:

- 343 retries in one hour
- 157 retries during the first 100 seconds

## Documentation

- `ram` GO packages are documented on line: [https://godoc.org/github.com/BrunoReboul/ram](https://godoc.org/github.com/BrunoReboul/ram)
