package keymgmt

import (
	"strings"
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

func dev(uuid, addr, manufacture string) *model.Device {
	d := &model.Device{}
	d.UUID = uuid
	a := addr
	d.AddressUUID = &a
	d.Manufacture = manufacture
	return d
}

const (
	keyA = "0F1E2D3C4B5A69788796A5B4C3D2E1F0"
	keyB = "112233445566778899AABBCCDDEEFF00"
)

func TestFindDuplicateKeyOwner_NoDuplicate(t *testing.T) {
	fleet := []*model.Device{
		dev("u1", "5CC0D947", keyA),
		dev("u2", "C3C0A660", keyB),
	}
	// A third device with its own distinct key is fine.
	if owner, dup := FindDuplicateKeyOwner(fleet, "u3", "AABBCCDDEEFF00112233445566778899"); dup {
		t.Errorf("unexpected duplicate against %q", owner)
	}
}

func TestFindDuplicateKeyOwner_DetectsReuse(t *testing.T) {
	fleet := []*model.Device{
		dev("u1", "5CC0D947", keyA),
		dev("u2", "C3C0A660", keyB),
	}
	owner, dup := FindDuplicateKeyOwner(fleet, "u3", keyA)
	if !dup {
		t.Fatal("reusing another device's key must be reported")
	}
	if owner != "5CC0D947" {
		t.Errorf("owner = %q, want 5CC0D947", owner)
	}
}

// Editing a device and keeping its own key is not a duplicate — otherwise every
// save of an unchanged device would be rejected.
func TestFindDuplicateKeyOwner_SelfIsNotDuplicate(t *testing.T) {
	fleet := []*model.Device{dev("u1", "5CC0D947", keyA)}
	if owner, dup := FindDuplicateKeyOwner(fleet, "u1", keyA); dup {
		t.Errorf("a device keeping its own key must not clash with itself (got %q)", owner)
	}
}

// Same key written in different shapes is still the same key.
func TestFindDuplicateKeyOwner_NormalisesFormat(t *testing.T) {
	fleet := []*model.Device{dev("u1", "5CC0D947", keyA)}
	for _, variant := range []string{
		strings.ToLower(keyA),
		"0F-1E-2D-3C-4B-5A-69-78-87-96-A5-B4-C3-D2-E1-F0", // ESP32 style
		"  " + keyA + "  ",
	} {
		if _, dup := FindDuplicateKeyOwner(fleet, "u2", variant); !dup {
			t.Errorf("variant %q must be recognised as the same key", variant)
		}
	}
}

// An empty key means "use the shared key" — many devices legitimately do that,
// so it must never be reported as a duplicate.
func TestFindDuplicateKeyOwner_EmptyKeyIsNeverDuplicate(t *testing.T) {
	fleet := []*model.Device{
		dev("u1", "5CC0D947", ""),
		dev("u2", "C3C0A660", ""),
	}
	if owner, dup := FindDuplicateKeyOwner(fleet, "u3", ""); dup {
		t.Errorf("empty key must not clash (got %q)", owner)
	}
}

func TestFindDuplicateKeyOwner_IgnoresDevicesWithoutKeyOrAddress(t *testing.T) {
	noAddr := &model.Device{}
	noAddr.UUID = "u9"
	noAddr.Manufacture = keyA // same key but no address to report

	fleet := []*model.Device{nil, noAddr, dev("u1", "5CC0D947", "")}
	if _, dup := FindDuplicateKeyOwner(fleet, "u2", keyA); dup {
		t.Error("a device with no address cannot be named as the owner; must not report a duplicate")
	}
}
