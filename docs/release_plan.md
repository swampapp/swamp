# Towards the 1.0.0 public release

Complete all the items in the [Swamp 1.0 milestone](https://github.com/swampapp/swamp/milestone/1).

## Current status

The app is currently a proof of concept.

Swamp will go through a number of alpha releases before reaching its beta quality status.

Alpha status generally means it's missing important features and high severity or critical bugs are still present, usually as a result of proof of concept code being added. During the alpha period, on-disk data structures/format may also change, forcing the user to start reset the local state if no data migrations are added.

Once the app is considered stable enough and has no (known) high severity or critical bugs, Swamp will move to a beta status. Strong QA (manual or automated), polishing and documentation will reign the beta period. Only fixes for existing bugs that don't require large code changes will be merged.

Progress is being tracked using the [the 1.0 milestone page](https://github.com/swampapp/swamp/milestone/1).

**From a user point of view**, the app is mostly functional, with a few missing features will be implemented before the 1.0 release:

* [A graphical user interface](https://github.com/swampapp/swamp/issues/1) to manage preferences
* [An updated indexer pane](https://github.com/swampapp/swamp/issues/2)
* [Cancel in-progress downloads](https://github.com/swampapp/swamp/issues/3)
* Option to [prevent listing duplicated files when searching](https://github.com/swampapp/swamp/issues/4)
* [Export downloaded files](https://github.com/swampapp/swamp/issues/5)
* [Quick start guide](https://github.com/swampapp/swamp/issues/6)

**From a data safety perspective**, Swamp doesn't change your Restic repository, it can work with read-only repositories.

Swamp stores three types of data locally:

* The index: can be deleted and recreated if necessary, any time, so data quality bugs that may happen when indexing (`~/.local/share/com.github.swampapp/repositories/<repo ID>/index`)
* Downloaded files: Swamp can download repository files locally, when instructed to do so. The worst thing that can happen is that you may need to re-download them, if something goes wrong (shared across repos in `~/.local/share/com.github.swampapp/downloads`).
* Tags: this is the only user data that can't be recreated right now. Currently stored in `~/.local/share/com.github.swampapp/repositories/<repo ID>/tags.db`, can be safely backed up when the app is closed.

So the tags database is the only user data worth saving if you need to start from scratch to test things.

**From a developer point of view:**

* The component model needs to be solidified and documented, including:
  * How to add new components with optional Glade files
  * Standardize the way we want components to notify observers of state changes. I've tried three different approaches, none of them fully backed and/or automatically tested currently
* I'm reasonably happy with the way the repository has been structured, document it.
* Not necessarily for 1.0, but I should at least find a way to automate the testing of the UI parts, maybe using something like [robotgo](https://github.com/go-vgo/robotgo).
* Consolidate settings/config/keychain management in a single package (`resticsettings`, `settings` and `config` packages should be merged into one.)
* Better error handling and error reporting throughout the entire code base. Most errors need to be surfaced correctly for the user to see. The status package and UI will probably need to be rewritten as a result.
* Most of the backend code needs unit/integration tests and CI needs to be added
* Backend code for the indexer (`indexer` package) and downloader (`downloader` package) needs to be refactored.
* Refactor [the queries package](https://github.com/swampapp/swamp/issues/7).
