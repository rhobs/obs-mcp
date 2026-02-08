// test-chart serves the timeseries chart UI in a test harness for local development.
// It simulates the MCP Apps protocol, letting you preview and tune the chart
// without connecting to a real MCP client or Prometheus instance.
//
// Usage:
//
//	go run ./cmd/test-chart
//
// Then open http://localhost:9199 in your browser.
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, harness)
	})
	http.HandleFunc("/chart", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, buildChartHTML())
	})

	addr := "127.0.0.1:9199"
	fmt.Fprintf(os.Stderr, "Chart test harness: http://%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func buildChartHTML() string {
	tmpl, err := os.ReadFile("pkg/mcp/ui/chart.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Run this command from the repo root:\n  go run ./cmd/test-chart\n\nerror: %v\n", err)
		os.Exit(1)
	}
	styles, _ := os.ReadFile("pkg/mcp/ui/styles.css")
	chartLib, _ := os.ReadFile("pkg/mcp/ui/chart.min.js")
	dateAdapter, _ := os.ReadFile("pkg/mcp/ui/date-adapter.js")
	app, _ := os.ReadFile("pkg/mcp/ui/app.js")

	r := strings.NewReplacer(
		"{{STYLES}}", string(styles),
		"{{CHART_LIB}}", string(chartLib),
		"{{DATE_ADAPTER}}", string(dateAdapter),
		"{{APP}}", string(app),
	)
	return r.Replace(string(tmpl))
}

const harness = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Chart Test Harness</title>
<style>
  :root {
    --bg: #f3f4f6; --text: #111827; --surface: #ffffff;
    --border: #d1d5db; --subtle: #6b7280;
  }
  [data-theme="dark"] {
    --bg: #111827; --text: #f3f4f6; --surface: #1f2937;
    --border: #374151; --subtle: #9ca3af;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: system-ui, -apple-system, sans-serif;
    background: var(--bg);
    color: var(--text);
    min-height: 100vh;
    transition: background 0.2s, color 0.2s;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 20px;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
    flex-wrap: wrap;
  }
  .toolbar h1 {
    font-size: 15px;
    font-weight: 600;
    margin-right: auto;
  }
  .btn-group {
    display: flex;
    border: 1px solid var(--border);
    border-radius: 8px;
    overflow: hidden;
  }
  .btn-group button {
    padding: 6px 14px;
    font-size: 13px;
    font-family: inherit;
    background: var(--surface);
    color: var(--text);
    border: none;
    border-right: 1px solid var(--border);
    cursor: pointer;
    transition: background 0.15s;
  }
  .btn-group button:last-child { border-right: none; }
  .btn-group button:hover { background: var(--bg); }
  .btn-group button.active {
    background: #2563eb; color: #fff;
  }
  button.action {
    padding: 6px 16px;
    font-size: 13px;
    font-family: inherit;
    background: #2563eb;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.15s;
  }
  button.action:hover { background: #1d4ed8; }
  select, input[type="text"] {
    padding: 6px 10px;
    font-size: 13px;
    font-family: inherit;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface);
    color: var(--text);
  }
  select { cursor: pointer; }
  input[type="text"] { width: 220px; }
  label {
    font-size: 13px;
    color: var(--subtle);
  }
  .frame-area {
    padding: 20px;
  }
  iframe {
    width: 100%;
    height: 0;
    border: none;
    display: block;
    border-radius: 4px;
    transition: height 0.2s;
  }
</style>
</head>
<body>

<div class="toolbar">
  <h1>Chart Test Harness</h1>

  <label>Theme</label>
  <div class="btn-group" id="theme-btns">
    <button class="active" data-theme="light">Light</button>
    <button data-theme="dark">Dark</button>
  </div>

  <label>Series</label>
  <select id="series-count">
    <option value="1">1</option>
    <option value="3">3</option>
    <option value="5" selected>5</option>
    <option value="10">10</option>
  </select>

  <label>Range</label>
  <select id="time-range">
    <option value="1800">30 min</option>
    <option value="7200" selected>2 hours</option>
    <option value="43200">12 hours</option>
    <option value="86400">24 hours</option>
    <option value="259200">3 days</option>
  </select>

  <label>Title</label>
  <input type="text" id="title-input" value="CPU Usage by Pod (Last 2 Hours)" placeholder="Chart title (optional)">

  <button class="action" onclick="sendData()">Send Data</button>
  <button class="action" onclick="clearData()" style="background:#dc2626">Clear</button>
</div>

<div class="frame-area">
  <iframe id="f" src="/chart"></iframe>
</div>

<script>
var dark = false;
var f = document.getElementById("f");

// Restore selections from URL params
(function() {
  var p = new URLSearchParams(window.location.search);
  if (p.has("theme")) {
    var t = p.get("theme");
    if (t === "dark" || t === "light") {
      dark = t === "dark";
      document.documentElement.setAttribute("data-theme", t);
      document.querySelectorAll("#theme-btns button").forEach(function(b) {
        b.classList.toggle("active", b.dataset.theme === t);
      });
    }
  }
  if (p.has("series")) {
    var s = document.getElementById("series-count");
    if (s.querySelector('option[value="' + p.get("series") + '"]')) s.value = p.get("series");
  }
  if (p.has("range")) {
    var r = document.getElementById("time-range");
    if (r.querySelector('option[value="' + p.get("range") + '"]')) r.value = p.get("range");
  }
  if (p.has("title")) {
    document.getElementById("title-input").value = p.get("title");
  }
})();

function updateURL() {
  var p = new URLSearchParams();
  p.set("theme", dark ? "dark" : "light");
  p.set("series", document.getElementById("series-count").value);
  p.set("range", document.getElementById("time-range").value);
  var title = document.getElementById("title-input").value.trim();
  if (title) p.set("title", title);
  history.replaceState(null, "", "?" + p.toString());
}

// Theme switching
document.getElementById("theme-btns").addEventListener("click", function(e) {
  var btn = e.target.closest("button");
  if (!btn) return;
  var theme = btn.dataset.theme;
  dark = theme === "dark";

  // Update button states
  this.querySelectorAll("button").forEach(function(b) { b.classList.remove("active"); });
  btn.classList.add("active");

  // Update harness theme
  document.documentElement.setAttribute("data-theme", theme);

  // Send to chart iframe
  f.contentWindow.postMessage({
    jsonrpc: "2.0",
    method: "ui/notifications/host-context-changed",
    params: { theme: theme }
  }, "*");

  updateURL();
});

// Handle MCP Apps lifecycle messages from iframe
window.addEventListener("message", function(e) {
  var m = e.data;
  if (!m || !m.jsonrpc) return;
  if (m.method === "ui/initialize") {
    f.contentWindow.postMessage({
      jsonrpc: "2.0",
      id: m.id,
      result: { hostContext: { theme: dark ? "dark" : "light" } }
    }, "*");
  }
  if (m.method === "ui/notifications/initialized") {
    sendData();
  }
  if (m.method === "ui/notifications/size-changed") {
    var h = (m.params && m.params.height) || 0;
    f.style.height = h > 0 ? "calc(100vh - 120px)" : "0";
  }
});

// Sample metric definitions
var METRICS = [
  { name: "container_cpu_usage_seconds_total", labels: { namespace: "openshift-monitoring", pod: "prometheus-k8s-0" }, base: 4.5, amp: 1.0 },
  { name: "container_cpu_usage_seconds_total", labels: { namespace: "openshift-monitoring", pod: "prometheus-k8s-1" }, base: 0.7, amp: 0.3 },
  { name: "container_cpu_usage_seconds_total", labels: { namespace: "openshift-monitoring", pod: "node-exporter-k62n8" }, base: 0.5, amp: 0.4 },
  { name: "container_cpu_usage_seconds_total", labels: { namespace: "openshift-monitoring", pod: "node-exporter-cxgcx" }, base: 0.4, amp: 0.3 },
  { name: "container_cpu_usage_seconds_total", labels: { namespace: "openshift-monitoring", pod: "kube-apiserver-ip-10-0-7-32" }, base: 0.6, amp: 0.5 },
  { name: "container_memory_working_set_bytes", labels: { namespace: "openshift-monitoring", pod: "prometheus-k8s-0" }, base: 2.1e9, amp: 3e8 },
  { name: "container_memory_working_set_bytes", labels: { namespace: "openshift-monitoring", pod: "alertmanager-main-0" }, base: 1.5e9, amp: 2e8 },
  { name: "container_memory_working_set_bytes", labels: { namespace: "openshift-monitoring", pod: "thanos-querier-0" }, base: 8e8, amp: 1.5e8 },
  { name: "node_cpu_seconds_total", labels: { instance: "ip-10-0-7-32:9100", mode: "idle" }, base: 95, amp: 3 },
  { name: "node_cpu_seconds_total", labels: { instance: "ip-10-0-8-15:9100", mode: "idle" }, base: 92, amp: 5 },
];

function sendData() {
  var count = parseInt(document.getElementById("series-count").value);
  var range = parseInt(document.getElementById("time-range").value);
  var now = Math.floor(Date.now() / 1000);
  var start = now - range;
  var step = Math.max(15, Math.floor(range / 120)); // ~120 data points

  var selected = METRICS.slice(0, count);
  var result = selected.map(function(s) {
    var metric = Object.assign({ __name__: s.name }, s.labels);
    var base = s.base;
    var amp = s.amp;
    var values = [];
    for (var t = start; t <= now; t += step) {
      var progress = (t - start) / range;
      var trend = Math.sin(progress * Math.PI * 2) * amp;
      var noise = (Math.random() - 0.5) * amp * 0.3;
      var v = base + trend + noise;
      values.push([t, String(Math.max(0, v))]);
    }
    return { metric: metric, values: values };
  });

  var queryName = selected[0] ? selected[0].name : "up";
  var query = "topk(" + count + ", sum(rate(" + queryName + "[5m])) by (pod, namespace))";
  var title = document.getElementById("title-input").value.trim();

  updateURL();

  // Send tool-input (query + optional title)
  var toolArgs = { query: query };
  if (title) toolArgs.title = title;
  f.contentWindow.postMessage({
    jsonrpc: "2.0",
    method: "ui/notifications/tool-input",
    params: { arguments: toolArgs }
  }, "*");

  // Send tool-result (data)
  var sc = { resultType: "matrix", result: result };
  f.contentWindow.postMessage({
    jsonrpc: "2.0",
    method: "ui/notifications/tool-result",
    params: { structuredContent: sc }
  }, "*");
}

function clearData() {
  f.contentWindow.postMessage({
    jsonrpc: "2.0",
    method: "ui/resource-teardown",
    params: {}
  }, "*");
}
</script>
</body>
</html>
`
