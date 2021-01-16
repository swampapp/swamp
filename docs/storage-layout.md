# Storage layout

## Configuration directory

Lives under `$HOME/.config/com.githubapp.swamp`.

## Data directory

Lives under `$HOME/.local/share/com.github.swampapp`.

❯ ls ~/.local/share/com.github.swampapp/repositories/bef9c329b3c046c682d97bb4da793a3a2a31e2dc0ae43616a40cd93cef2a8563
index  tags.db

### The index

Swamp uses the experimental [bluge](https://github.com/blugelabs/bluge) (via [rindex](https://github.com/rubiojr/rindex)) to index and search indexed files.

There's one index per repository managed by Swamp:

**~/.local/share/com.github.swampapp/repositories**: holds repository indices. Each subdirectory represents a Restic repository and is named after Restic's repository ID.

```
❯ ls ~/.local/share/com.github.swampapp/repositories/bef9c329b3c046c682d97bb4da793a3a2a31e2dc0ae43616a40cd93cef2a8563
index  tags.db
```

The `index` directory holds the Bluge index.

### Tags database

`tags.db` is a [LevelDB](https://github.com/syndtr/goleveldb) key/value database.
