package catalog

import (
	"encoding/json"
	"strings"
)

const ShieldAttributeKey = "Шильдик"

// BadgeFromAttributesJSON извлекает шильдик из JSON attributes фида.
func BadgeFromAttributesJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return ""
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return ""
	}
	return BadgeFromAttributesMap(attrs)
}

// BadgeFromAttributesMap извлекает шильдик из map атрибутов.
func BadgeFromAttributesMap(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	if val := strings.TrimSpace(attrs[ShieldAttributeKey]); val != "" {
		return val
	}
	for key, val := range attrs {
		if strings.EqualFold(strings.TrimSpace(key), ShieldAttributeKey) {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// HasNewBadge сообщает, есть ли шильдик NEW.
func HasNewBadge(attrsJSON string) bool {
	return strings.EqualFold(strings.TrimSpace(BadgeFromAttributesJSON(attrsJSON)), "NEW")
}

// MergeShieldBadgeIntoAttributes проставляет шильдик в JSON attributes, если его ещё нет.
func MergeShieldBadgeIntoAttributes(attrsJSON, badge string) string {
	badge = strings.TrimSpace(badge)
	if badge == "" {
		return attrsJSON
	}
	if strings.EqualFold(strings.TrimSpace(BadgeFromAttributesJSON(attrsJSON)), badge) {
		return attrsJSON
	}

	attrs := map[string]string{}
	raw := strings.TrimSpace(attrsJSON)
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &attrs)
	}
	attrs[ShieldAttributeKey] = badge
	out, err := json.Marshal(attrs)
	if err != nil {
		return attrsJSON
	}
	return string(out)
}
