# e2e

Crude, down to earth e2e testing for systemk. Requires a running "k3s-server" and systemk. Once you
have those running you can test for yourself with: `go test -v -tags e2e` in the "e2e" directory.
Note that "k3s" must be in your PATH.

The tests wrap "k3s kubectl" to apply config, mostly to be able to use YAML in the tests *and* to
not pull down all of kubernetes for client-go access to the kubernetes API. The downside is that we
sometimes need to parse kubectl output instead of getting it from the API.

These tests are also run via a GitHub workflow.
