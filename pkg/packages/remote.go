package packages

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// fetch fetches a package from the URL pointed at in pkg. The URL must be given with a scheme (http:// or https://).
func fetch(pkg, version string) (string, error) {
	c := new(http.Client)
	c.Timeout = 240 * time.Second

	pkg = "https://" + pkg
	klog.Infof("Fetching from %s", pkg)
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

// Clean checks the string pkg and returns the package name. Everything up to the first _ is the package name after
// the scheme (https:// or http://).
// If the string is an absolute path, the base is returned as the package name.
// On error the pkg is returned as-is.
func Clean(pkg string) string {
	if path.IsAbs(pkg) {
		return path.Base(pkg)
	}

	if !strings.HasPrefix(pkg, "http://") && !strings.HasPrefix(pkg, "https://") {
		return pkg
	}
	u, err := url.Parse(pkg)
	if err != nil {
		return pkg
	}
	deb := path.Base(u.Path)
	i := strings.Index(deb, "_")
	if i < 2 {
		return pkg
	}
	return deb[:i]
}
