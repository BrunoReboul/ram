// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package services structure

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

*/
package services
