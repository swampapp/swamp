type:video +demoscene
type:audio
type:document
type:image

type:video +size:>100mb -> size > 100mb && (ext:mp4 || ext:mkv || ext:avi)

* Videos that were modified during the last 7 days

+modified:recently type:video

* Files bigger than 1GB that were modified (or created) during the last 7 days

+modified:recently +size:>1GB

added:recently
added:today
added:yesterday

* explain "at least one" queries (without +), i.e. ext:pdf ext:doc
