package alertmanagement

import "testing"

func TestToolsetIncludesListAlerts(t *testing.T) {
	names := map[string]struct{}{}
	for _, st := range (&Toolset{}).GetTools(nil) {
		names[st.Tool.Name] = struct{}{}
	}
	for _, want := range []string{
		"list_alerts",
		"list_alert_rules",
		"preview_alert_rule",
		"create_alert_rule",
		"update_alert_rule",
		"delete_alert_rules",
	} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing tool %s", want)
		}
	}
}
