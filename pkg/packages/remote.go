package packages

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// fetch fetches a package from the URL pointed at in pkg. The URL must be given without a packager
// prefix (deb://, or arch://). etc.
func fetch(pkg, version string) (string, error) {
	c := new(http.Client)
	c.Timeout = 240 * time.Second

	pkg = "https://" + pkg
	log.Printf("Fetching from %s", pkg)
	resp, err := c.Get(pkg)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got non 200 status code for %s: %d", pkg, resp.StatusCode)
	}

	tmppkg, err := ioutil.TempFile(os.TempDir(), "package*.deb")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmppkg, resp.Body)
	if err != nil {
		return "", err
	}
	return tmppkg.Name(), nil
}
