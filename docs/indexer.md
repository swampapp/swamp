# Indexer

Swamp indexes Restic repositories using a background process, `swampd`.

The indexing process runs periodically every 30 minutes, and the application communicates with it using a UNIX socket (`$HOME/.local/share/com.github.swampd/indexing.sock`).

The indexing process exposes 3 HTTP endpoints:

## /status

Reports (almost) real-time JSON indexing stats:

```
curl -s --unix-socket ~/.local/share/com.github.swampapp/indexing.sock http://localhost/stats | jq
  {
  "Mismatch": 0,
  "ScannedNodes": 0,
  "IndexedFiles": 0,
  "ScannedSnapshots": 0,
  "AlreadyIndexed": 0,
  "ScannedFiles": 0,
  "Errors": null,
  "LastMatch": "",
  "MissingSnapshots": 0,
  "SnapshotFiles": null,
  "CurrentSnapshotFiles": 0,
  "CurrentSnapshotTotalFiles": 0,
  "TotalSnapshots": 0
  }
```

## /kill

Gracefully stops the indexing process.


```
curl -s --unix-socket ~/.local/share/com.github.swampapp/indexing.sock http://localhost/kill | jq

"sutting down"
```

## /ping

Check if the indexing process has started.

```
curl -s --unix-socket ~/.local/share/com.github.swampapp/indexing.sock http://localhost/ping  jq

"pong"
```

Note that if the indexing process died without being able to gracefully shutdown, an old (dead) socket file may be around, so checking for the presence of the socket file is not enough to figure out if the indexing process is running.
