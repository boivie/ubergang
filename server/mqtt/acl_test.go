package mqtt

import (
	"boivie/ubergang/server/models"
	"testing"
)

func TestFilterCovers(t *testing.T) {
	tests := []struct {
		allowed, sub string
		want         bool
	}{
		// Exact and single-level wildcards.
		{"a/b", "a/b", true},
		{"a/b", "a/c", false},
		{"a/b", "a/b/c", false},
		{"a/b/c", "a/b", false},
		{"a/+/c", "a/x/c", true},
		{"a/+/c", "a/x/y/c", false},
		{"a/+", "a/x", true},
		{"a/+", "a", false},

		// Multi-level wildcard, including the parent level itself.
		{"a/#", "a/b/c", true},
		{"a/#", "a", true},
		{"a/#", "ab", false},
		{"#", "anything/at/all", true},
		{"#", "a", true},

		// Filters must match from the first level: no substring matches.
		{"eye/#", "homeassistant/eye/x", false},
		{"shellies/announce", "x/shellies/announce", false},
		{"shellies/announce", "shellies/announce/x", false},
		{"shellies/announce", "shellies/announceX", false},
		// "." and other regex characters are literal.
		{"nibe/a.b", "nibe/aXb", false},
		{"nibe/a.b", "nibe/a.b", true},

		// "$" topics aren't matched by first-level wildcards.
		{"#", "$SYS/broker/uptime", false},
		{"+/broker", "$SYS/broker", false},
		{"$SYS/#", "$SYS/broker/uptime", true},

		// Subscriptions covered by a wider allowed filter.
		{"zigbee2mqtt/#", "zigbee2mqtt/#", true},
		{"zigbee2mqtt/#", "zigbee2mqtt/bridge/state", true},
		{"zigbee2mqtt/#", "zigbee2mqtt/+/set", true},
		{"#", "homeassistant/#", true},
		{"#", "+/status", true},
		{"a/+", "a/+", true},
		{"a/+/c", "a/+/c", true},
		// ... but not ones reaching beyond it.
		{"zigbee2mqtt/bridge/state", "zigbee2mqtt/#", false},
		{"a/b", "a/+", false},
		{"a/+", "a/#", false},
		{"a/b/#", "a/#", false},
		{"homeassistant/status", "homeassistant/#", false},

		// Malformed allowed filter ("#" not last) never matches.
		{"a/#/b", "a/x/b", false},
		{"", "a", false},
		{"a", "", false},
	}
	for _, tc := range tests {
		if got := filterCovers(tc.allowed, tc.sub); got != tc.want {
			t.Errorf("filterCovers(%q, %q) = %v, want %v", tc.allowed, tc.sub, got, tc.want)
		}
	}
}

func TestACLWithProfileValues(t *testing.T) {
	acl, err := NewACL(
		&models.MqttClient{Id: "shelly-sanglampa", Values: map[string]string{"SERIALNO": "26748E"}},
		&models.MqttProfile{
			AllowPublish: []string{"shellies/announce", "shellies/shellyplug-s-$SERIALNO/#"},
			AllowSubscribe: []string{
				"shellies/command",
				"shellies/shellyplug-s-$SERIALNO/relay/0/command",
				"cmnd/DVES_$SERIALNO_fb/#",
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	publish := map[string]Classification{
		"shellies/announce":                          ALLOWED,
		"shellies/shellyplug-s-26748E/relay/0":       ALLOWED,
		"shellies/shellyplug-s-26748E/relay/0/power": ALLOWED,
		"shellies/shellyplug-s-FFFFFF/relay/0":       BLOCKED, // another device
		"homeassistant/shellies/announce":            BLOCKED,
		"shellies/shellyplug-s-26748E/+":             BLOCKED, // wildcards in a topic name
	}
	for topic, want := range publish {
		if got := acl.ValidatePublishTopic(topic); got != want {
			t.Errorf("publish %q = %s, want %s", topic, ClassificationToString(got), ClassificationToString(want))
		}
	}

	subscribe := map[string]Classification{
		"shellies/command": ALLOWED,
		"shellies/shellyplug-s-26748E/relay/0/command": ALLOWED,
		"cmnd/DVES_26748E_fb/#":                        ALLOWED,
		"cmnd/DVES_26748E_fb/power":                    ALLOWED,
		"shellies/#":                                   BLOCKED,
		"shellies/shellyplug-s-FFFFFF/relay/0/command": BLOCKED,
		"shellies/shellyplug-s-26748E/relay/+/command": BLOCKED,
	}
	for filter, want := range subscribe {
		if got := acl.ValidateSubscribeTopic(filter); got != want {
			t.Errorf("subscribe %q = %s, want %s", filter, ClassificationToString(got), ClassificationToString(want))
		}
	}
}

func TestFullAccessProfile(t *testing.T) {
	acl, err := NewACL(&models.MqttClient{Id: "hass"},
		&models.MqttProfile{AllowPublish: []string{"#"}, AllowSubscribe: []string{"#"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{"homeassistant/status", "zigbee2mqtt/lamp/set", "a"} {
		if acl.ValidatePublishTopic(topic) != ALLOWED {
			t.Errorf("publish %q blocked with full access", topic)
		}
	}
	for _, filter := range []string{"#", "homeassistant/#", "+/+/state", "zigbee2mqtt/lamp"} {
		if acl.ValidateSubscribeTopic(filter) != ALLOWED {
			t.Errorf("subscribe %q blocked with full access", filter)
		}
	}
}
