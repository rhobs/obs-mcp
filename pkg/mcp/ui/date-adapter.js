// Minimal Chart.js date adapter using native Date/Intl APIs.
// Avoids bundling date-fns (~30KB) or luxon (~68KB).
(function() {
  var FORMATS = {
    datetime: { hour: "2-digit", minute: "2-digit", second: "2-digit", month: "short", day: "numeric" },
    millisecond: { hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 3 },
    second: { hour: "2-digit", minute: "2-digit", second: "2-digit" },
    minute: { hour: "2-digit", minute: "2-digit" },
    hour: { hour: "2-digit", minute: "2-digit" },
    day: { month: "short", day: "numeric" },
    week: { month: "short", day: "numeric" },
    month: { month: "short", year: "numeric" },
    quarter: { month: "short", year: "numeric" },
    year: { year: "numeric" }
  };

  Chart._adapters._date.override({
    formats: function() { return FORMATS; },
    parse: function(value) {
      if (value === null || value === undefined) return null;
      return typeof value === "number" ? value : new Date(value).getTime();
    },
    format: function(time, fmt) {
      var opts = typeof fmt === "string" ? (FORMATS[fmt] || {}) : fmt;
      return new Intl.DateTimeFormat(undefined, opts).format(new Date(time));
    },
    add: function(time, amount, unit) {
      var d = new Date(time);
      switch (unit) {
        case "millisecond": d.setTime(d.getTime() + amount); break;
        case "second": d.setSeconds(d.getSeconds() + amount); break;
        case "minute": d.setMinutes(d.getMinutes() + amount); break;
        case "hour": d.setHours(d.getHours() + amount); break;
        case "day": d.setDate(d.getDate() + amount); break;
        case "week": d.setDate(d.getDate() + amount * 7); break;
        case "month": d.setMonth(d.getMonth() + amount); break;
        case "quarter": d.setMonth(d.getMonth() + amount * 3); break;
        case "year": d.setFullYear(d.getFullYear() + amount); break;
      }
      return d.getTime();
    },
    diff: function(max, min, unit) {
      var ms = max - min;
      switch (unit) {
        case "millisecond": return ms;
        case "second": return ms / 1000;
        case "minute": return ms / 60000;
        case "hour": return ms / 3600000;
        case "day": return ms / 86400000;
        case "week": return ms / 604800000;
        case "month":
          return (new Date(max).getFullYear() - new Date(min).getFullYear()) * 12
            + new Date(max).getMonth() - new Date(min).getMonth();
        case "quarter":
          return ((new Date(max).getFullYear() - new Date(min).getFullYear()) * 12
            + new Date(max).getMonth() - new Date(min).getMonth()) / 3;
        case "year": return new Date(max).getFullYear() - new Date(min).getFullYear();
        default: return ms;
      }
    },
    startOf: function(time, unit) {
      var d = new Date(time);
      switch (unit) {
        case "second": d.setMilliseconds(0); break;
        case "minute": d.setSeconds(0, 0); break;
        case "hour": d.setMinutes(0, 0, 0); break;
        case "day": d.setHours(0, 0, 0, 0); break;
        case "week":
          d.setDate(d.getDate() - d.getDay());
          d.setHours(0, 0, 0, 0);
          break;
        case "month": d.setDate(1); d.setHours(0, 0, 0, 0); break;
        case "quarter":
          d.setMonth(d.getMonth() - d.getMonth() % 3, 1);
          d.setHours(0, 0, 0, 0);
          break;
        case "year": d.setMonth(0, 1); d.setHours(0, 0, 0, 0); break;
      }
      return d.getTime();
    }
  });
})();
