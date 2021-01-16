# Storage layout

## Configuration directory

Lives under `$HOME/.config/com.githubapp.swamp`.

## Data directory

Lives under `$HOME/.local/share/com.github.swampapp`.

```
❯ ls ~/.local/share/com.github.swampapp/repositories/bef9c329b3c046c682d97bb4da793a3a2a31e2dc0ae43616a40cd93cef2a8563
index  tags.db
```

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

### Downloads directory

Every file downloaded by Swamp is stored in `~/.local/share/com.github.swampapp/downloads`.

Each file downloaded is named after the file ID (a SHA256 of the file content + filename).

If a file's `file ID` is `7c228af37e56d6255f4f4c3ad5bce9b9a827475557bdc96524f69bd7f333fdec`, it'll be stored in `~/.local/share/com.github.swampapp/downloads/7c/7c228af37e56d6255f4f4c3ad5bce9b9a827475557bdc96524f69bd7f333fdec`.

**Note:** this will change before the final 1.0 release, as Swamp will use the BHash (file's content hash) as the downloaded file name.
