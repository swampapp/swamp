# Towards the 1.0.0 public release

## Current status

The app is currently a proof of concept.

From a user point of view, the app is mostly functional, with a few missing features will be implemented before the 1.0 release:

* [A graphical user interface](https://github.com/swampapp/swamp/issues/1) to manage preferences
* [An updated indexer pane](https://github.com/swampapp/swamp/issues/2)
* [Cancel in-progress downloads](https://github.com/swampapp/swamp/issues/3)
* Option to [prevent listing duplicated files when searching](https://github.com/swampapp/swamp/issues/4)
* [Export downloaded files](https://github.com/swampapp/swamp/issues/5)
* [Quick start guide](https://github.com/swampapp/swamp/issues/6)

From a developer point of view:

* The component model needs to be solidified and documented, including:
  * How to add new components with optional Glade files
  * Standardize the way we want components to notify observers of state changes. I've tried three different approaches, none of them fully backed and/or automatically tested currently
* I'm reasonably happy with the way the repository has been structured, document it.
* Not necessarily for 1.0, but I should at least find a way to automate the testing of the UI parts, maybe using something like [robotgo](https://github.com/go-vgo/robotgo).
* Consolidate settings/config/keychain management in a single package (`resticsettings`, `settings` and `config` packages should be merged into one.)
* Better error handling and error reporting throughout the entire code base. Most errors need to be surfaced correctly for the user to see. The status package and UI will probably need to be rewritten as a result.
* Most of the backend code needs unit/integration tests and CI needs to be added
* Backend code for the indexer (`indexer` package) and downloader (`downloader` package) needs to be refactored.
