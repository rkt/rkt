package scf

import (
	"fmt"

	"testing"
)

/* load an exec-group manifest */
func TestLoadManifest(t *testing.T) {
	json :=	[]byte(`{"SEEVersion": "1","UID": "1111C000-A000-4444-AABB-AAAAAAAAAAAA","Group": { "example.com/data-downloader-1.0.0": {"ID": "sha1-3e86b59982e49066c5d813af1c2e2579cbf573de","Before": ["example.com/ourapp-1.0.0"]}, "example.com/ourapp-1.0.0": {"ID": "sha1-277205b3ae3eb3a8e042a62ae46934b470e431ac","Before": ["example.com/logbackup-1.0.0"]}, "example.com/logbackup-1.0.0": {"ID": "sha1-86298e1fdb95ec9a45b5935504e26ec29b8feffa"}},"Volumes": {"database": {"Path": "/opt/tenant1/database"}}}`)

	egm, err := loadManifest(json)
	if err != nil {
		t.Fatal("Failed to load egm json: %v", err)
	}

	fmt.Printf("Group: \"%s\"\n", egm.UID);

	for i, u := range egm.Units {
		fmt.Printf(" Unit %v: ID: \"%s\"\n", i, u.ID);
		for _, req := range u.Prereqs {
			fmt.Printf("  Prereq: \"%s\"\n", req);
		}
	}
}


/* load an exec-group manifest and constituent runnable container units */
func TestLoadExecGroup(t *testing.T) {
	eg, err := LoadExecGroup("examples/execgroup-skel")
	if err != nil {
		t.Fatal("Failed to load execgroup: %v", err)
	}

	fmt.Printf("Group: \"%s\"\n", eg.Manifest.UID)
	for _, u := range eg.Units {
		fmt.Printf(" Unit \"%s\"", u.Name)
		if eg.Manifest.Units[u.Name].Prereqs != nil {
			fmt.Printf(" Before:")
			for _, pre := range eg.Manifest.Units[u.Name].Prereqs {
				fmt.Printf(" \"%s\"", pre)
			}
		}
		fmt.Printf("\n")
	}
}
