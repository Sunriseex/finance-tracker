package services

import (
	"expvar"
	"strconv"
	"strings"
)

var authEventsTotal = expvar.NewMap("capitalflow_auth_events_total")

func recordAuthEventMetric(eventType string, success bool, reason string) {
	authEventsTotal.Add(authEventMetricKey(eventType, success, reason), 1)
}

func authEventMetricKey(eventType string, success bool, reason string) string {
	eventType = metricLabelValue(eventType, "unknown")
	reason = metricLabelValue(reason, "none")
	return "event_type=" + eventType + ",success=" + strconv.FormatBool(success) + ",reason=" + reason
}

func metricLabelValue(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	return strings.NewReplacer(",", "_", "=", "_", " ", "_").Replace(value)
}
