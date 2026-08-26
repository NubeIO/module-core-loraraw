package pkg

import (
	"fmt"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-loraraw/keymgmt"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
)

// ProvisionStatus is the non-secret view of a device's key state. It never
// carries key material — the only place a key is ever returned is the response
// to BeginProvision, where the operator needs it to load the device.
type ProvisionStatus struct {
	DeviceUUID  string `json:"device_uuid"`
	AddressUUID string `json:"address_uuid,omitempty"`
	Mode        string `json:"mode"`
	State       string `json:"provision_state"`
	HasOwnKey   bool   `json:"has_own_key"`
	HasPending  bool   `json:"has_pending_key"`
}

// BeginProvisionResponse carries the freshly generated key. This is the single
// point in the system where key material crosses the API boundary: the operator
// (or a provisioning tool) needs it to write the key into the device over its
// wired link. It is never logged.
type BeginProvisionResponse struct {
	ProvisionStatus
	// NewKey is the generated key as uppercase hex, ready for the device's
	// AT+AES= (STM32) or param_set 0x0220 (ESP32) command.
	NewKey string `json:"new_key"`
	// Instructions tells the operator exactly what to do next, including the
	// firmware quirks discovered on real hardware.
	Instructions []string `json:"instructions"`
}

// deviceWithMetaTags fetches a device including its meta tags, which is where
// key lifecycle state lives.
func (m *Module) deviceWithMetaTags(uuid string) (*model.Device, error) {
	dev, err := m.grpcMarshaller.GetDevice(uuid, &nmodule.Opts{Args: &nargs.Args{WithMetaTags: true}})
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, fmt.Errorf("device %s not found", uuid)
	}
	return dev, nil
}

// applyPlan persists a lifecycle transition: the live key (when the plan
// changes it) and the meta tags.
//
// dev is the device as currently stored. UpdateDevice replaces the record
// rather than patching it, and the framework rejects a body with an empty Name
// ("name cannot be empty"), so the key change is applied on top of the existing
// record instead of on a bare struct.
func (m *Module) applyPlan(dev *model.Device, plan keymgmt.Plan) error {
	deviceUUID := dev.UUID
	if plan.Manufacture != "" {
		body := *dev // copy: do not mutate the caller's device
		body.Manufacture = plan.Manufacture
		// Meta tags are persisted separately below; sending them here would
		// duplicate the write.
		body.MetaTags = nil
		if _, err := m.grpcMarshaller.UpdateDevice(deviceUUID, &body); err != nil {
			return fmt.Errorf("update device key: %w", err)
		}
	}
	if len(plan.MetaTags) > 0 {
		if err := m.grpcMarshaller.UpsertDeviceMetaTags(deviceUUID, plan.MetaTags, nil); err != nil {
			return fmt.Errorf("update key state: %w", err)
		}
	}
	return nil
}

func statusOf(dev *model.Device, res keymgmt.Resolution) ProvisionStatus {
	s := ProvisionStatus{
		DeviceUUID: dev.UUID,
		Mode:       string(res.Mode),
		State:      string(res.State),
		HasOwnKey:  dev.Manufacture != "",
		HasPending: res.PendingKey != nil,
	}
	if dev.AddressUUID != nil {
		s.AddressUUID = *dev.AddressUUID
	}
	return s
}

// GetProvisionStatus reports a device's key mode and lifecycle state.
func (m *Module) GetProvisionStatus(deviceUUID string) (*ProvisionStatus, error) {
	dev, err := m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}
	res, err := m.resolveDeviceKey(dev)
	if err != nil {
		return nil, err
	}
	defer res.Zero()
	st := statusOf(dev, res)
	return &st, nil
}

// BeginProvision generates a new per-device key and stages it.
//
// The device's live key is deliberately NOT replaced yet: until the device has
// taken and saved the new key it is still using the old one, and overwriting
// now would cut contact. The staged key is promoted by ConfirmProvision once
// the device proves it holds it (doc 08 section 8.2).
func (m *Module) BeginProvision(deviceUUID string) (*BeginProvisionResponse, error) {
	dev, err := m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}

	newKeyHex, err := keymgmt.GenerateKeyHex()
	if err != nil {
		return nil, err
	}

	plan, err := keymgmt.BeginProvision(dev, newKeyHex)
	if err != nil {
		return nil, err
	}
	if err := m.applyPlan(dev, plan); err != nil {
		return nil, err
	}

	// Deliberately logs the device, never the key.
	log.Infof("provisioning: staged a new key for device %s (state=PENDING)", deviceUUID)

	dev, err = m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}
	res, err := m.resolveDeviceKey(dev)
	if err != nil {
		return nil, err
	}
	defer res.Zero()

	return &BeginProvisionResponse{
		ProvisionStatus: statusOf(dev, res),
		NewKey:          newKeyHex,
		Instructions: []string{
			"STM32 (Droplet, Optical Power Meter): send AT+AES=<new_key> then AT+SAVE. Key is plain hex with NO separators.",
			"ESP32 (FGA Gen2): send param_set 0x0220 <new_key_dashed> — the ESP32 console REQUIRES dash-separated hex (AA-BB-CC-...), plain hex is parsed as the wrong length and rejected.",
			"Send the command SLOWLY, ~20ms per character: the AT loop reads one character at a time and drops input sent as a burst.",
			"AT+SAVE is REQUIRED — AT+AES= only writes RAM, so without it the key is lost on reset.",
			"The serial port may drop briefly during AT+SAVE while the MCU writes flash; that is expected.",
			"Then confirm with POST /api/devices/<uuid>/key/confirm once the device is transmitting under the new key.",
		},
	}, nil
}

// ConfirmProvision promotes the staged key to the device's live key. Call it
// once the device has been loaded and is transmitting successfully.
func (m *Module) ConfirmProvision(deviceUUID string) (*ProvisionStatus, error) {
	dev, err := m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}
	plan, err := keymgmt.ConfirmProvision(dev)
	if err != nil {
		return nil, err
	}
	if err := m.applyPlan(dev, plan); err != nil {
		return nil, err
	}
	log.Infof("provisioning: device %s promoted to its own key (state=ACTIVE)", deviceUUID)
	return m.GetProvisionStatus(deviceUUID)
}

// AbortProvision drops the staged key and leaves the device on its current one.
func (m *Module) AbortProvision(deviceUUID string) (*ProvisionStatus, error) {
	dev, err := m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}
	plan, err := keymgmt.AbortProvision(dev)
	if err != nil {
		return nil, err
	}
	if err := m.applyPlan(dev, plan); err != nil {
		return nil, err
	}
	log.Infof("provisioning: aborted for device %s, staged key discarded", deviceUUID)
	return m.GetProvisionStatus(deviceUUID)
}

// RetireDevice tombstones a device's key state.
func (m *Module) RetireDevice(deviceUUID string) (*ProvisionStatus, error) {
	dev, err := m.deviceWithMetaTags(deviceUUID)
	if err != nil {
		return nil, err
	}
	plan, err := keymgmt.RetireDevice(dev)
	if err != nil {
		return nil, err
	}
	if err := m.applyPlan(dev, plan); err != nil {
		return nil, err
	}
	log.Infof("provisioning: device %s retired", deviceUUID)
	return m.GetProvisionStatus(deviceUUID)
}
