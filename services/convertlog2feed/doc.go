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
Package convertlog2feed convert log entry messsage from pubsub into feed compatible format pubusub message

Instances

Some.

At least one to convert gsuite admin logs, related to groups members and groups settings, from GCP Cloud Audit Logs at organization level.

Output

Publish pubsub message into several topics, e.g. gci-grouMembers and gci-groupSettings.

Cardinality

One one: one log message one feed message.

Notes

- AdminSDK activity logs do not reference the group ID only the email, that is a mutable attribute. A cache in firestore is used to handle this.

*/
package convertlog2feed
