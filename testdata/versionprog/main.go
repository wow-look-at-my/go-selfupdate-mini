// versionprog is a tiny program used by TestEmbeddedVersionViaLDFlags to
// verify that ldflags-injected versions reach CurrentVersion() at runtime.
// Lives under testdata so it is excluded from `go build ./...` and never
// ships as part of the library.
package main

import (
	"fmt"

	selfupdate "github.com/wow-look-at-my/go-selfupdate-mini"
)

func main() {
	fmt.Print(selfupdate.CurrentVersion())
}
