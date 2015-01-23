package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/coreos/rocket/metadata"
	rktpath "github.com/coreos/rocket/path"
)

func registerContainer(c *Container, ip net.IP) error {
	cmf, err := os.Open(rktpath.ContainerManifestPath(c.Root))
	if err != nil {
		return fmt.Errorf("failed opening runtime manifest: %v", err)
	}
	defer cmf.Close()

	pth := fmt.Sprintf("/containers/?ip=%v", ip.To4().String())
	if err := httpRequest("POST", pth, cmf); err != nil {
		return fmt.Errorf("failed to register container with metadata svc: %v", err)
	}

	uid := c.Manifest.UUID.String()
	for _, app := range c.Manifest.Apps {
		ampath := rktpath.ImageManifestPath(c.Root, app.ImageID)
		amf, err := os.Open(ampath)
		if err != nil {
			fmt.Errorf("failed reading app manifest %q: %v", ampath, err)
		}
		defer amf.Close()

		if err := registerApp(uid, app.Name.String(), amf); err != nil {
			fmt.Errorf("failed to register app with metadata svc: %v", err)
		}
	}

	return nil
}

func unregisterContainer(c *Container) error {
	pth := path.Join("/containers", c.Manifest.UUID.String())
	return httpRequest("DELETE", pth, nil)
}

func registerApp(uuid, app string, r io.Reader) error {
	pth := path.Join("/containers", uuid, app)
	return httpRequest("PUT", pth, r)
}

func httpRequest(method, pth string, body io.Reader) error {
	uri := metadata.SvcPrvURL() + pth
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return err
	}

	cli := http.Client{}

	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("%v %v returned %v", method, pth, resp.StatusCode)
	}

	return nil
}
