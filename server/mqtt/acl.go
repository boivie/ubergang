package mqtt

import (
	"boivie/ubergang/server/models"
	"log"
	"regexp"
	"strings"
)

type Classification int

const (
	ALLOWED Classification = iota
	BLOCKED
)

type ACL struct {
	AllowPublish   []*regexp.Regexp
	BlockPublish   []*regexp.Regexp
	AllowSubscribe []string
	BlockSubscribe []string
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

	makeRegex := func(pattern string) (*regexp.Regexp, error) {
		pattern = strings.ReplaceAll(pattern, "+", "[^/]+")
		pattern = strings.ReplaceAll(pattern, "/#", "/.+")
		return regexp.Compile(pattern)
	}

	c := &ACL{}

	for _, topic := range clientProfile.AllowPublish {
		re, err := makeRegex(subst(topic))
		if err != nil {
			return nil, err
		}
		c.AllowPublish = append(c.AllowPublish, re)
	}
	for _, topic := range clientProfile.AllowSubscribe {
		c.AllowSubscribe = append(c.AllowSubscribe, subst(topic))
	}
	return c, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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

func (a *ACL) ValidatePublishTopic(topic string) Classification {
	for _, pattern := range a.AllowPublish {
		if pattern.MatchString(topic) {
			return ALLOWED
		}
	}
	return BLOCKED
}

func (a *ACL) FilterValidSubscribeTopics(topics []string) (allowed []string) {
	for _, topic := range topics {
		if contains(a.AllowSubscribe, topic) {
			allowed = append(allowed, topic)
		}
	}
	return
}

func (a *ACL) ValidateSubscribeTopic(topic string) Classification {
	for _, t := range a.AllowSubscribe {
		if t == topic {
			return ALLOWED
		}
	}
	return BLOCKED
}
