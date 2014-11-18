package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containers/standard/schema"
	"github.com/coreos-inc/rkt/rkt"
)

// ContainerRuntimeManifest + AppManifests encapsulator
type Container struct {
	Manifest *schema.ContainerRuntimeManifest
	Apps     map[string]*schema.AppManifest
}

// prep a container manifest
func prepManifest(blob []byte) (*schema.ContainerRuntimeManifest, error) {
	cm := &schema.ContainerRuntimeManifest{}

	if err := json.Unmarshal(blob, cm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group manifest: %v", err)
	}

	/*
		// TODO(jonboulle): remove depends..
		// ensure each app.Depends refers to valid Apps[key]
		for _, app := range cm.Apps {
			if app.Depends != nil {
				for _, b := range app.Depends {
					if _, ok := cm.Apps[b]; !ok {
						return nil, fmt.Errorf("invalid depends: %s", b)
					}
				}
			}
		}
	*/

	// TODO any further necessary validation, like detecting circular Depends?
	// I don't think anything we do here is rkt-specific, such validation likely belongs in the standard unmarshal if possible
	return cm, nil
}

// load a stage0-prepared container and all its app's manifests beneath $root/stage1/opt/stage2/$apphash
func LoadContainer(root string) (*Container, error) {
	c := &Container{
		Apps: make(map[string]*schema.AppManifest),
	}

	buf, err := ioutil.ReadFile(ContainerManifestPath(false))
	if err != nil {
		return nil, fmt.Errorf("failed reading container manifest: %v", err)
	}

	c.Manifest, err = prepManifest(buf)
	if err != nil {
		return nil, fmt.Errorf("failed preparing container manifest: %v", err)
	}

	for _, app := range c.Manifest.Apps {
		ampath := rkt.AppManifestPath(root, app.ImageID.String())
		buf, err := ioutil.ReadFile(ampath)
		if err != nil {
			return nil, fmt.Errorf("failed reading app manifest %q: %v", ampath, err)
		}

		am := &schema.AppManifest{}
		if err = json.Unmarshal(buf, am); err != nil {
			return nil, fmt.Errorf("failed to unmarshal app manifest %q: %v", ampath, err)
		}
		name := am.Name.String()
		if _, ok := c.Apps[name]; ok {
			return nil, fmt.Errorf("got multiple definitions for app: %s", name)
		}
		c.Apps[name] = am
	}

	return c, nil
}
