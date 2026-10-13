package pkg

import (
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
)

// The firmware version pair reported by the UART adapter: point 5 carries the packed release, point 6 the rc number.
func TestUARTPointConfigFirmwareVersionPair(t *testing.T) {
	cases := map[string]string{"5": "Firmware Version", "6": "Release Candidate"}
	for id, name := range cases {
		cfg := getUARTPointConfig(id)
		if cfg == nil {
			t.Fatalf("point %s: no config", id)
		}
		if cfg.Name != name {
			t.Fatalf("point %s: name %q, want %q", id, cfg.Name, name)
		}
		if cfg.DataType != "30" {
			t.Fatalf("point %s: data type %q, want 30 (UINT_8)", id, cfg.DataType)
		}
		if cfg.WriteMode != datatype.ReadOnly {
			t.Fatalf("point %s: write mode %q, want read-only", id, cfg.WriteMode)
		}
	}
}

// UVP-8 is the adapter's setting level, writable from Rubix as a boolean.
func TestUARTPointConfigAdminSettingLevel(t *testing.T) {
	cfg := getUARTPointConfig("8")
	if cfg == nil {
		t.Fatal("point 8: no config")
	}
	if cfg.Name != "Admin Setting Level" {
		t.Fatalf("point 8: name %q, want Admin Setting Level", cfg.Name)
	}
	if cfg.DataType != "38" {
		t.Fatalf("point 8: data type %q, want 38 (BOOL)", cfg.DataType)
	}
	if cfg.WriteMode != datatype.WriteAlways {
		t.Fatalf("point 8: write mode %q, want write-always", cfg.WriteMode)
	}
}
