# Travis build serializer

Serializes `push` builds to a given branch without setting a concurrency limit on the entire repository. Skips old builds in favor of newer ones.

## Goals and approach

### One build at a time

When a build starts, it checks if it should proceed. If not, it exits by cancelling itself.

For this to work, every build needs to be able to look at the API state and make the same decision about which build should proceed.

Our solution is to select the build that has the **earliest `started_at` time** of any running (`started`) build. When a build observes itself to be first in this ordering, it will remain first until it exits -- and it will always have been first from the perspective of all other running builds.

### Builds run in queued order

Even if we're sure only one build runs at a time, it's conceivable that an older build will be delayed long enough for a newer build to run and finish.

Travis builds are created with monotonically-increasing `ID`s. In order to prevent older builds from racing past newer ones, each build checks that it has a **higher `ID`** than any finished (`passed`, `failed`, or `errored`) build.

This still allows restarting the most recently finished build.

### Newest build will eventually run

Builds cancel themselves when it's not their turn to run.

When a running build finishes (successfully or not), it looks at the most recently queued build (i.e. highest `ID`) and restarts it if its state is `canceled`. Intervening builds are skipped for efficiency.

It is possible for a build to cancel itself at exactly the wrong moment: before the build it's waiting for has finished, but _after_ that build has looked for -- and not found -- a `canceled` build to restart.

The next queued build will run without incident, so this will only be a momentary headache for active repositories. If it becomes a real problem, we may consider having the older build wait until the newest queued build is `canceled`, so it can be restarted.
