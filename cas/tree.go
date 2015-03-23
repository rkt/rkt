package cas

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/pkg/aci"
)

const (
	hashfilename     = "hash"
	renderedfilename = "rendered"
)

// TreeStore represents a store of rendered ACIs
// The image's key becomes the name of the directory containing the rendered aci.
type TreeStore struct {
	path string
}

// Write renders the ACI with the provided key in the treestore
// Write, to avoid having a rendered ACI with old stale files, requires that
// the destination directory doesn't exist (usually Remove should be called
// before Write)
func (ts *TreeStore) Write(key string, ds *Store) error {
	treepath := filepath.Join(ts.path, key)
	fi, _ := os.Stat(treepath)
	if fi != nil {
		return fmt.Errorf("treestore: path %s already exists", treepath)
	}
	imageID, err := types.NewHash(key)
	if err != nil {
		return fmt.Errorf("treestore: cannot convert key to imageID: %v", err)
	}
	err = aci.RenderACIWithImageID(*imageID, treepath, ds)
	if err != nil {
		return fmt.Errorf("treestore: cannot render aci: %v", err)
	}
	// before creating the "rendered" flag file we need to ensure that all data is fsynced
	err = filepath.Walk(treepath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsDir() && !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		err = f.Sync()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("treestore: failed to sync data: %v", err)
	}
	// Create rendered file
	f, err := os.Create(filepath.Join(treepath, renderedfilename))
	if err != nil {
		return fmt.Errorf("treestore: failed to write rendered file: %v", err)
	}
	f.Close()

	df, err := os.Open(treepath)
	if err != nil {
		return err
	}
	defer df.Close()
	err = df.Sync()
	if err != nil {
		return fmt.Errorf("treestore: failed to sync tree store directory: %v", err)
	}
	return nil
}

// Remove cleans the directory for the specified key
func (ts *TreeStore) Remove(key string) error {
	treepath := filepath.Join(ts.path, key)
	// If tree path doesn't exist we're done
	_, err := os.Stat(treepath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("treestore: failed to open tree store directory: %v", err)
	}

	renderedFilePath := filepath.Join(treepath, renderedfilename)
	// The "rendered" flag file should be the firstly removed file. So if
	// the removal ends with some error leaving some stale files IsRendered()
	// will return false.
	_, err = os.Stat(renderedFilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !os.IsNotExist(err) {
		err := os.Remove(renderedFilePath)
		// Ensure that the treepath directory is fsynced after removing the
		// "rendered" flag file
		f, err := os.Open(treepath)
		if err != nil {
			return fmt.Errorf("treestore: failed to open tree store directory: %v", err)
		}
		defer f.Close()
		err = f.Sync()
		if err != nil {
			return fmt.Errorf("treestore: failed to sync tree store directory: %v", err)
		}
	}
	return os.RemoveAll(treepath)
}

// IsRendered checks if the tree store is fully rendered
func (ts *TreeStore) IsRendered(key string) (bool, error) {
	// if the "rendered" flag file exists, assume that the store is already
	// fully rendered.
	treepath := filepath.Join(ts.path, key)
	_, err := os.Stat(filepath.Join(treepath, renderedfilename))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetPath returns the absolute path of the treestore for the specified key.
// It doesn't ensure that the path exists and is fully rendered. This should
// be done calling IsRendered()
func (ts *TreeStore) GetPath(key string) string {
	return filepath.Join(ts.path, key)
}

// GetRootFS returns the absolute path of the rootfs for the specified key.
// It doesn't ensure that the rootfs exists and is fully rendered. This should
// be done calling IsRendered()
func (ts *TreeStore) GetRootFS(key string) string {
	return filepath.Join(ts.GetPath(key), "rootfs")
}
