# systemk-volume

This is a helper binary that chowns specific paths *before* the main units start taking over. This
helps to move as much state from systemk into systemd.

Every pod gets two extra units, one that runs before and one that runs after: they will be called:

* prestart: chowning dirs etc.
* postexit: when the pod is stopped/unloaded this runs.

Note we *can* use shell scripts to perform the same actions, but using a Go binary gives more
flexibilty/portablity and safety.
