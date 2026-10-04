package pkg

import (
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// Legacy UVP points keep their numeric config key and numeric address id.
func TestSetNewPointFieldsUARTKeepsUVPMapping(t *testing.T) {
	for _, io := range []string{"UVP-1", "uvp-1"} {
		p := &model.Point{}
		setNewPointFieldsUART(&model.Device{}, p, io)
		if p.Name != "Communication Status" {
			t.Fatalf("%s: name %q, want Communication Status", io, p.Name)
		}
		if p.IoNumber != io {
			t.Fatalf("%s: io number %q must be kept verbatim", io, p.IoNumber)
		}
		if p.AddressID == nil || *p.AddressID != 1 {
			t.Fatalf("%s: address id %v, want 1", io, p.AddressID)
		}
	}
}

// DVP points are keyed by type so DVP-1 does not inherit UVP-1's config.
func TestSetNewPointFieldsUARTMapsDVPByType(t *testing.T) {
	p := &model.Point{}
	setNewPointFieldsUART(&model.Device{}, p, "DVP-1")
	if p.Name != "Prohibit All Operation" {
		t.Fatalf("DVP-1 name %q, want Prohibit All Operation", p.Name)
	}
	if p.DataType != "30" {
		t.Fatalf("DVP-1 data type %q, want 30 (UINT_8)", p.DataType)
	}
	if p.WriteMode != datatype.WriteAlways {
		t.Fatalf("DVP-1 write mode %q, want %q", p.WriteMode, datatype.WriteAlways)
	}
	if p.EnableWriteable == nil || !*p.EnableWriteable {
		t.Fatalf("DVP-1 must be writeable")
	}
	if p.AddressID != nil {
		t.Fatalf("DVP-1 address id %d must not reuse the UVP numeric address space", *p.AddressID)
	}

	ro := &model.Point{}
	setNewPointFieldsUART(&model.Device{}, ro, "dvp-8")
	if ro.Name != "Prohibition Of Operation" {
		t.Fatalf("dvp-8 name %q, want Prohibition Of Operation", ro.Name)
	}
	if ro.WriteMode != datatype.ReadOnly {
		t.Fatalf("dvp-8 write mode %q, want %q", ro.WriteMode, datatype.ReadOnly)
	}
	if ro.EnableWriteable == nil || *ro.EnableWriteable {
		t.Fatalf("dvp-8 must be read-only")
	}
}

// A DVP id without a table entry falls back to the generic defaults.
func TestSetNewPointFieldsUARTUnknownDVPFallsBack(t *testing.T) {
	p := &model.Point{}
	setNewPointFieldsUART(&model.Device{}, p, "DVP-9")
	if p.Name == "" || p.Name == "Communication Status" {
		t.Fatalf("DVP-9 name %q, want generic fallback name", p.Name)
	}
	if p.WriteMode != datatype.ReadOnly {
		t.Fatalf("DVP-9 write mode %q, want read-only fallback", p.WriteMode)
	}
}
