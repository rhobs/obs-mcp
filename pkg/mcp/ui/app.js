// MCP Apps timeseries chart application.
// Handles MCP Apps lifecycle, data transformation, and Chart.js rendering.
(function() {
  "use strict";

  // ===== State =====
  var hostContext = null;
  var chartInstance = null;
  var lastResult = null;
  var queryString = null;
  var titleString = null;
  var currentUnit = null;
  var requestId = 1;

  // ===== Color Palettes (PatternFly v6 multi-color ordered chart colors) =====
  var LIGHT_COLORS = [
    "#0066cc", "#63993d", "#37a3a3", "#ca6c0f", "#9e4a06",
    "#004d99", "#3d7317", "#147878", "#b98412", "#96640f"
  ];
  var DARK_COLORS = [
    "#4394e5", "#87bb62", "#63bdbd", "#ffcc17", "#f5921b",
    "#92c5f9", "#afdc8f", "#9ad8d8", "#ffe072", "#f8ae54"
  ];

  // ===== Theme Utilities =====
  function isDark() {
    return hostContext && hostContext.theme === "dark";
  }

  function getColors() {
    return isDark() ? DARK_COLORS : LIGHT_COLORS;
  }

  function applyTheme() {
    document.documentElement.setAttribute("data-theme", isDark() ? "dark" : "light");
    if (chartInstance && lastResult) {
      renderChart(lastResult);
    }
  }

  // ===== JSON-RPC Messaging =====
  function send(msg) {
    window.parent.postMessage(msg, "*");
  }

  function sendRequest(method, params) {
    var id = requestId++;
    send({ jsonrpc: "2.0", method: method, id: id, params: params || {} });
    return id;
  }

  function sendNotification(method, params) {
    send({ jsonrpc: "2.0", method: method, params: params || {} });
  }

  // ===== Data Transformation =====
  function formatLabel(metric) {
    if (!metric || Object.keys(metric).length === 0) return "{}";
    var name = metric.__name__ || "";
    var rest = Object.keys(metric)
      .filter(function(k) { return k !== "__name__"; })
      .sort()
      .map(function(k) { return k + '="' + metric[k] + '"'; })
      .join(", ");
    if (name && rest) return name + "{" + rest + "}";
    if (name) return name;
    return "{" + rest + "}";
  }

  function buildDatasets(result) {
    if (!result || result.length === 0) return null;

    var palette = getColors();
    var datasets = [];

    for (var i = 0; i < result.length; i++) {
      var series = result[i];
      var values = series.values;
      if (!values || values.length === 0) continue;

      var points = [];
      for (var j = 0; j < values.length; j++) {
        points.push({
          x: values[j][0] * 1000,
          y: parseFloat(values[j][1])
        });
      }

      var color = palette[i % palette.length];
      datasets.push({
        label: formatLabel(series.metric),
        data: points,
        borderColor: color,
        backgroundColor: color,
        borderWidth: 1.5,
        pointRadius: 0,
        pointHitRadius: 8,
        pointHoverRadius: 4,
        pointHoverBackgroundColor: color,
        pointHoverBorderColor: isDark() ? "#242424" : "#ffffff",
        pointHoverBorderWidth: 2,
        tension: 0.15,
        fill: false,
        spanGaps: false
      });
    }

    return datasets.length > 0 ? datasets : null;
  }

  function truncate(str, max) {
    return str.length > max ? str.substring(0, max - 1) + "\u2026" : str;
  }

  // ===== Chart Rendering =====
  function renderChart(result) {
    lastResult = result;

    var wrapper = document.getElementById("chart-wrapper");
    var canvas = document.getElementById("chart-canvas");

    if (chartInstance) {
      chartInstance.destroy();
      chartInstance = null;
    }

    var datasets = buildDatasets(result);
    if (!datasets) {
      wrapper.innerHTML = '<div class="no-data">No data to display</div>';
      return;
    }

    // Restore canvas if it was replaced by no-data message
    if (!wrapper.querySelector("canvas")) {
      wrapper.innerHTML = '<canvas id="chart-canvas"></canvas>';
      canvas = document.getElementById("chart-canvas");
    }

    // Compute data minimum to decide Y-axis floor
    var dataMin = Infinity;
    for (var di = 0; di < datasets.length; di++) {
      for (var pi = 0; pi < datasets[di].data.length; pi++) {
        if (datasets[di].data[pi].y < dataMin) dataMin = datasets[di].data[pi].y;
      }
    }

    var dark = isDark();
    var gridColor = dark ? "rgba(255, 255, 255, 0.06)" : "rgba(0, 0, 0, 0.06)";
    var tickColor = dark ? "#9ca3af" : "#6b7280";

    var isPercent = currentUnit === "percent";

    var yScale = {
      min: dataMin >= 0 ? 0 : undefined,
      max: isPercent ? 100 : undefined,
      ticks: {
        color: tickColor,
        font: { size: 12, weight: "500", family: "system-ui, sans-serif" },
        padding: 8,
        maxTicksLimit: 8,
        callback: function(value) {
          if (isPercent) return Math.round(value);
          if (Math.abs(value) >= 1e9) return (value / 1e9).toFixed(1) + "G";
          if (Math.abs(value) >= 1e6) return (value / 1e6).toFixed(1) + "M";
          if (Math.abs(value) >= 1e3) return (value / 1e3).toFixed(1) + "k";
          if (Math.abs(value) < 0.01 && value !== 0) return value.toExponential(1);
          if (Math.abs(value) < 1) return value.toPrecision(3);
          return value.toFixed(2);
        }
      },
      grid: {
        color: gridColor,
        lineWidth: 1.5,
        drawTicks: false
      },
      border: {
        display: false
      }
    };

    // Show unit label above the chart
    var unitLabel = document.getElementById("unit-label");
    if (isPercent) {
      unitLabel.textContent = "Percent (%)";
      unitLabel.classList.add("visible");
    } else {
      unitLabel.classList.remove("visible");
    }

    chartInstance = new Chart(canvas, {
      type: "line",
      data: { datasets: datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: {
          mode: "index",
          intersect: false
        },
        hover: {
          mode: "index",
          intersect: false
        },
        scales: {
          x: {
            type: "time",
            time: {
              tooltipFormat: "datetime",
              displayFormats: {
                second: "minute",
                minute: "minute",
                hour: "hour",
                day: "day",
                week: "day",
                month: "month"
              }
            },
            ticks: {
              color: tickColor,
              font: { size: 12, family: "system-ui, sans-serif" },
              maxRotation: 0,
              autoSkip: true,
              maxTicksLimit: Math.max(3, Math.floor(wrapper.clientWidth / 120))
            },
            grid: {
              display: false
            },
            border: {
              display: false
            }
          },
          y: yScale
        },
        plugins: {
          legend: {
            display: false
          },
          tooltip: {
            backgroundColor: dark ? "rgba(55, 65, 81, 0.7)" : "rgba(31, 41, 55, 0.7)",
            titleColor: "#f9fafb",
            bodyColor: "#e5e7eb",
            titleFont: { size: 12, family: "system-ui, sans-serif" },
            bodyFont: { size: 11, family: "system-ui, sans-serif" },
            padding: { top: 8, bottom: 8, left: 12, right: 12 },
            cornerRadius: 8,
            displayColors: true,
            boxWidth: 8,
            boxHeight: 8,
            usePointStyle: true,
            caretPadding: 6,
            callbacks: {
              label: function(ctx) {
                var value = ctx.parsed.y;
                var formatted;
                if (Math.abs(value) < 0.01 && value !== 0) {
                  formatted = value.toExponential(2);
                } else if (Math.abs(value) < 100) {
                  formatted = value.toFixed(2);
                } else {
                  formatted = value.toLocaleString(undefined, { maximumFractionDigits: 1 });
                }
                var maxLen = Math.max(20, Math.floor(wrapper.clientWidth / 8));
                return " " + truncate(ctx.dataset.label, maxLen) + ": " + formatted;
              }
            }
          }
        },
        layout: {
          padding: { top: 4, right: 8, bottom: 0, left: 0 }
        }
      }
    });
  }

  // ===== Message Handler =====
  window.addEventListener("message", function(e) {
    var msg = e.data;
    if (!msg || !msg.jsonrpc) return;

    // Response to ui/initialize request
    if (msg.id === 1 && msg.result) {
      hostContext = msg.result.hostContext || {};
      applyTheme();
      sendNotification("ui/notifications/initialized");
      return;
    }

    // Tool input: capture query string and title
    if (msg.method === "ui/notifications/tool-input") {
      var input = msg.params || {};
      var args = input.arguments || input.input || input;
      var titleEl = document.getElementById("chart-title");
      if (args.title) {
        titleString = args.title;
        titleEl.textContent = titleString;
        titleEl.classList.add("visible");
      } else {
        titleString = null;
        titleEl.textContent = "";
        titleEl.classList.remove("visible");
      }
      if (args.query) {
        queryString = args.query;
        document.getElementById("query-value").textContent = queryString;
        document.getElementById("query-display").classList.add("visible");
      }
      return;
    }

    // Tool result: render chart
    if (msg.method === "ui/notifications/tool-result") {
      var params = msg.params || {};
      var sc = params.structuredContent;
      if (sc && sc.result) {
        // Also check for query in structured content as fallback
        if (sc.query && !queryString) {
          queryString = sc.query;
          document.getElementById("query-value").textContent = queryString;
          document.getElementById("query-display").classList.add("visible");
        }
        currentUnit = sc.unit || null;
        renderChart(sc.result);
      }
      return;
    }

    // Theme change
    if (msg.method === "ui/notifications/host-context-changed") {
      hostContext = msg.params || hostContext;
      applyTheme();
      return;
    }

    // Cleanup
    if (msg.method === "ui/resource-teardown") {
      if (chartInstance) {
        chartInstance.destroy();
        chartInstance = null;
      }
      lastResult = null;
      queryString = null;
      titleString = null;
      currentUnit = null;
      return;
    }
  });

  // ===== Resize Observer =====
  var resizeObserver = new ResizeObserver(function(entries) {
    var card = entries[0].target;
    var style = getComputedStyle(card);
    var marginW = parseFloat(style.marginLeft) + parseFloat(style.marginRight);
    var marginH = parseFloat(style.marginTop) + parseFloat(style.marginBottom);
    sendNotification("ui/notifications/size-changed", {
      width: Math.round(card.offsetWidth + marginW),
      height: Math.round(card.offsetHeight + marginH)
    });
  });
  resizeObserver.observe(document.getElementById("card"));

  // ===== Start MCP Apps Lifecycle =====
  sendRequest("ui/initialize", {
    appCapabilities: {
      availableDisplayModes: ["inline"]
    }
  });
})();
