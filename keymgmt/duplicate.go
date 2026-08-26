package keymgmt

import (
	"strings"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// NormaliseKeyHex puts a key into one comparable form: uppercase, no dashes, no
// spaces. The same key can be typed several ways - the ESP32 console prints
// dash-separated hex, the STM32 AT command wants it plain - and all of them
// must compare equal.
func NormaliseKeyHex(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return strings.ToUpper(s)
}

// FindDuplicateKeyOwner reports whether keyHex is already in use by a device
// other than selfUUID, and if so which one (by address).
//
// Reusing one key across devices silently defeats per-device keying: the whole
// point is that compromising one device does not expose the others. It is easy
// to do by accident - pasting the same value, or a documentation example, into
// several Device Key boxes.
//
// An empty key is never a duplicate: it means "use the shared key", which many
// devices legitimately do.
func FindDuplicateKeyOwner(devices []*model.Device, selfUUID, keyHex string) (string, bool) {
	want := NormaliseKeyHex(keyHex)
	if want == "" {
		return "", false
	}
	for _, d := range devices {
		if d == nil || d.UUID == selfUUID || d.Manufacture == "" {
			continue
		}
		// Without an address there is nothing useful to report back to the
		// operator, so such a record cannot be named as the owner.
		if d.AddressUUID == nil || *d.AddressUUID == "" {
			continue
		}
		if NormaliseKeyHex(d.Manufacture) == want {
			return *d.AddressUUID, true
		}
	}
	return "", false
}
