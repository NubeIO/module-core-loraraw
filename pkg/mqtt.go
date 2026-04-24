package pkg

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

// MQTTTopicPrefix is the base topic name used for all publishes from this module.
const MQTTTopicPrefix = "module-core-loraraw"

// MQTTClient is a thin wrapper around paho.mqtt.golang that:
//   - keeps a connection to the local broker open
//   - retries the initial connect forever (until the broker is up)
//   - automatically reconnects after a connection loss
//   - silently no-ops publishes while disconnected (so the rest of the module
//     keeps working even if the broker is down).
type MQTTClient struct {
	client      mqtt.Client
	topicPrefix string
	statusTopic string
	mu          sync.RWMutex
}

// normalizeBroker ensures the broker URL has the tcp:// scheme paho expects.
// Accepts values like "127.0.0.1:1883" or "localhost:1883" and prefixes tcp://.
func normalizeBroker(b string) string {
	b = strings.TrimSpace(b)
	if b == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(b), "tcp://") {
		return b
	}
	return "tcp://" + b
}

// NewMQTTClient constructs and starts a new MQTT client. Connection is attempted
// asynchronously and will be retried indefinitely until it succeeds. The same
// client will also automatically reconnect on any future disconnect.
// Returns nil when broker is empty.
func NewMQTTClient(broker, clientID, username, password, topicPrefix string) *MQTTClient {
	broker = normalizeBroker(broker)
	if broker == "" {
		log.Warn("mqtt: broker is empty, MQTT publishing disabled")
		return nil
	}
	if topicPrefix == "" {
		topicPrefix = MQTTTopicPrefix
	}
	if clientID == "" {
		clientID = MQTTTopicPrefix
	}
	statusTopic := fmt.Sprintf("%s/status", topicPrefix)

	c := &MQTTClient{topicPrefix: topicPrefix, statusTopic: statusTopic}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5*time.Second).
		SetMaxReconnectInterval(30*time.Second).
		SetKeepAlive(30*time.Second).
		SetPingTimeout(10*time.Second).
		SetCleanSession(true).
		SetOrderMatters(false).
		// Last Will: if this module dies or the TCP link drops without a
		// clean disconnect, the broker publishes "offline" retained.
		SetWill(statusTopic, "offline", 0, true).
		SetOnConnectHandler(func(cl mqtt.Client) {
			log.Infof("mqtt: connected to broker %s", broker)
			// Announce ourselves as online (retained) so any subscriber
			// joining later sees the current state immediately.
			tok := cl.Publish(statusTopic, 0, true, "online")
			go func() {
				tok.Wait()
				if err := tok.Error(); err != nil {
					log.Warnf("mqtt: failed to publish online status: %v", err)
				}
			}()
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Warnf("mqtt: connection lost: %v (auto-reconnect enabled)", err)
		}).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			log.Info("mqtt: reconnecting...")
		})

	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	c.client = mqtt.NewClient(opts)

	// Connect in background. With SetConnectRetry(true) the paho client will
	// keep retrying until the broker becomes reachable.
	go func() {
		token := c.client.Connect()
		token.Wait()
		if err := token.Error(); err != nil {
			log.Warnf("mqtt: initial connect attempt failed (will keep retrying): %v", err)
		}
	}()

	return c
}

// publish marshals and publishes the payload at QoS 0. Any error is logged but
// not returned because the rest of the data flow must not be affected by MQTT
// failures.
func (c *MQTTClient) publish(topic string, payload interface{}) {
	if c == nil || c.client == nil {
		return
	}
	if !c.client.IsConnectionOpen() {
		log.Debugf("mqtt: skipping publish to %s, not connected", topic)
		return
	}

	var data []byte
	var err error
	switch p := payload.(type) {
	case string:
		data = []byte(p)
	case []byte:
		data = p
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			log.Errorf("mqtt: failed to marshal payload for %s: %v", topic, err)
			return
		}
	}

	log.Infof("mqtt: publishing topic=%s bytes=%d payload=%s", topic, len(data), string(data))
	token := c.client.Publish(topic, 0, false, data)
	go func() {
		token.Wait()
		if err := token.Error(); err != nil {
			log.Warnf("mqtt: publish failed topic=%s err=%v", topic, err)
		}
	}()
}

// PublishRaw publishes the raw uplink hex string to <prefix>/raw.
func (c *MQTTClient) PublishRaw(raw string) {
	if c == nil {
		return
	}
	topic := fmt.Sprintf("%s/raw", c.topicPrefix)
	log.Infof("mqtt: publishing raw uplink topic=%s len=%d", topic, len(raw))
	c.publish(topic, raw)
}

// PublishValues publishes the decoded values for a device to
// <prefix>/value as a JSON object of the form:
//
//	{"device_address_uuid": "...", "device_name": "...", "payload": {...}}
func (c *MQTTClient) PublishValues(addressUUID, deviceName string, values map[string]float64) {
	if c == nil || len(values) == 0 {
		return
	}
	topic := fmt.Sprintf("%s/value", c.topicPrefix)
	envelope := map[string]interface{}{
		"device_address_uuid": addressUUID,
		"device_name":         deviceName,
		"payload":             values,
	}
	log.Infof("mqtt: publishing decoded values topic=%s address=%s device=%s points=%d",
		topic, addressUUID, deviceName, len(values))
	c.publish(topic, envelope)
}

// Disconnect cleanly shuts down the MQTT client, publishing an "offline"
// status (retained) before tearing down the connection.
func (c *MQTTClient) Disconnect() {
	if c == nil || c.client == nil {
		return
	}
	if c.client.IsConnectionOpen() {
		tok := c.client.Publish(c.statusTopic, 0, true, "offline")
		tok.WaitTimeout(500 * time.Millisecond)
	}
	if c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}
