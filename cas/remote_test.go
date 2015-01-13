package cas

import (
	"io/ioutil"
	"testing"
)

func TestNewRemote(t *testing.T) {
	const (
		u1   = "https://example.com"
		u2   = "https://foo.com"
		data = "asdf"
	)
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	ds := NewStore(dir)

	// Create our first Remote, and simulate Store() to create index
	na := NewRemote(u1, "")
	na.BlobKey = data
	ds.WriteIndex(na)

	// Create a new remote w the same parameters, reading from index should be fine
	nb := NewRemote(u1, "")
	err = ds.ReadIndex(nb)
	if err != nil {
		t.Fatalf("unexpected error reading index: %v", err)
	}
	if nb.BlobKey != data {
		t.Fatalf("bad data returned from store: got %v, want %v", nb.BlobKey, data)
	}

	// Create a new remote with a different URI
	nc := NewRemote(u2, "")
	err = ds.ReadIndex(nc)
	// Should get an error, since the URI shouldn't be indexed
	if err == nil {
		t.Errorf("unexpected nil error reading index")
	}
	// Remote shouldn't be populated
	if nc.BlobKey != "" {
		t.Errorf("unexpected blob: got %v", nc.BlobKey)
	}
}
