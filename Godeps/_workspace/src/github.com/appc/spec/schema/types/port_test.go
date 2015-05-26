package types

import (
	"testing"
)

func TestGoodPort(t *testing.T) {
	p := Port{
		Port:  32456,
		Count: 100,
	}
	if err := p.assertValid(); err != nil {
		t.Errorf("good port assertion failed: %v", err)
	}
}

func TestBadPort(t *testing.T) {
	p := Port{
		Port: 88888,
	}
	if p.assertValid() == nil {
		t.Errorf("bad port asserted valid")
	}
}

func TestBadRange(t *testing.T) {
	p := Port{
		Port:  32456,
		Count: 45678,
	}
	if p.assertValid() == nil {
		t.Errorf("bad port range asserted valid")
	}
}
