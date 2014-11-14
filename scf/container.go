package main

import (
	"fmt"
	"io/ioutil"
	"encoding/json"

	"github.com/containers/standard/schema"
)


/* ContainerManifest + AppManifests encapsulator */
type Container struct {
	Manifest	*schema.ContainerManifest
	Apps		[]*schema.AppManifest
}


/* prep a container manifest */
func prepManifest(blob []byte) (*schema.ContainerManifest, error) {
	cm := &schema.ContainerManifest{}

	err := json.Unmarshal(blob, cm)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal group manifest: %v", err)
	}

	/* ensure each app.Before refers to valid Apps[key] */
	for _, app := range cm.Apps {
		if app.Before != nil {
			for _, b := range app.Before {
				if _, ok := cm.Apps[b]; !ok {
					return nil, fmt.Errorf("Invalid before: %s", b)
				}
			}
		}
	}

	/* TODO any further necessary validation, like detecting circular Befores? */
	/* I don't think anything we do here is rkt-specific, such validation likely belongs in the standard unmarshal if possible */
	return cm, nil
}


/* load an stage0-prepared container and all its app's manifests beneath $root/stage1/opt/stage2/esc($name)/apps/$name */
func LoadContainer() (*Container, error) {
	c := &Container{}

	buf, err := ioutil.ReadFile(ContainerManifestPath())
	if err != nil {
		return nil, fmt.Errorf("Failed reading container manifest: %v", err)
	}

	c.Manifest, err = prepManifest(buf)
	if err != nil {
		return nil, fmt.Errorf("Failed preparing container manifest: %v", err)
	}

	for name, _ := range c.Manifest.Apps {
		ampath := AppManifestPath(name)
		buf, err := ioutil.ReadFile(ampath)
		if err != nil {
			return nil, fmt.Errorf("Failed reading app manifest \"%s\": %v", ampath, err)
		}

		am := &schema.AppManifest{}
		err = json.Unmarshal(buf, am)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshal app manifest \"%s\": %v", ampath, err)
		}
		c.Apps = append(c.Apps, am)
	}

	return c, nil
}
