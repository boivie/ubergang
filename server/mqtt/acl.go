package mqtt

import (
	"boivie/ubergang/server/models"
	"log"
	"strings"
)

type Classification int

const (
	ALLOWED Classification = iota
	BLOCKED
)

// ACL holds a client's profile topics, with the client's values ($ID,
// $SERIALNO, ...) substituted. Entries are MQTT topic filters: "+" matches
// one level and a trailing "#" matches any number of levels, including none.
type ACL struct {
	AllowPublish   []string
	AllowSubscribe []string
}

func NewACL(clientConfig *models.MqttClient, clientProfile *models.MqttProfile) (*ACL, error) {
	variables := make(map[string]string)
	variables["$ID"] = clientConfig.Id
	for k, v := range clientConfig.Values {
		variables["$"+k] = v
	}

	subst := func(s string) string {
		for k, v := range variables {
			s = strings.ReplaceAll(s, k, v)
		}
		return s
	}

	c := &ACL{}
	for _, topic := range clientProfile.AllowPublish {
		c.AllowPublish = append(c.AllowPublish, subst(topic))
	}
	for _, topic := range clientProfile.AllowSubscribe {
		c.AllowSubscribe = append(c.AllowSubscribe, subst(topic))
	}
	return c, nil
}

func ClassificationToString(c Classification) string {
	switch c {
	case ALLOWED:
		return "allowed"
	case BLOCKED:
		return "blocked"
	default:
		log.Fatalf("Unhandled classification: %v", c)
		return ""
	}
}

// ValidatePublishTopic allows a topic name if it matches one of the allowed
// publish filters in full.
func (a *ACL) ValidatePublishTopic(topic string) Classification {
	if strings.ContainsAny(topic, "+#") {
		return BLOCKED // wildcards aren't valid in topic names
	}
	for _, filter := range a.AllowPublish {
		if filterCovers(filter, topic) {
			return ALLOWED
		}
	}
	return BLOCKED
}

// ValidateSubscribeTopic allows a subscription filter if every topic it can
// match is also matched by one of the allowed subscribe filters, so a
// subscription never reaches beyond what the profile allows.
func (a *ACL) ValidateSubscribeTopic(filter string) Classification {
	for _, allowed := range a.AllowSubscribe {
		if filterCovers(allowed, filter) {
			return ALLOWED
		}
	}
	return BLOCKED
}

// filterCovers reports whether every topic matched by sub (a topic name or a
// topic filter) is also matched by the topic filter allowed.
func filterCovers(allowed, sub string) bool {
	if allowed == "" || sub == "" {
		return false
	}
	a := strings.Split(allowed, "/")
	s := strings.Split(sub, "/")

	// Wildcards at the first level don't match topics starting with "$"
	// (MQTT 3.1.1 section 4.7.2), such as $SYS/...
	if strings.HasPrefix(s[0], "$") && (a[0] == "#" || a[0] == "+") {
		return false
	}

	for i, al := range a {
		if al == "#" {
			return i == len(a)-1 // "#" is only valid as the last level
		}
		if i >= len(s) {
			return false
		}
		sl := s[i]
		switch {
		case sl == "#":
			return false // sub matches deeper topics that allowed doesn't
		case al == "+":
			// Matches any single level, literal or "+".
		case sl != al:
			return false
		}
	}
	return len(s) == len(a)
}
