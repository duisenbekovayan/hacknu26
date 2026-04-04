/* global Chart */
(function () {
  const MAX_POINTS = 240;
  const trainInput = document.getElementById("trainId");
  const connPill = document.getElementById("connPill");
  const healthValue = document.getElementById("healthValue");
  const healthGrade = document.getElementById("healthGrade");
  const healthRing = document.getElementById("healthRing");
  const healthCategory = document.getElementById("healthCategory");
  const factorsEl = document.getElementById("factors");
  const metricsEl = document.getElementById("metrics");
  const motorsEl = document.getElementById("motors");
  const alertsEl = document.getElementById("alerts");
  const lastTs = document.getElementById("lastTs");
  const mapDot = document.getElementById("mapDot");
  const mapCaption = document.getElementById("mapCaption");
  const btnTheme = document.getElementById("btnTheme");

  const buffers = {
    labels: [],
    speed: [],
    coolant: [],
    health: [],
  };

  let charts = { speed: null, temp: null, health: null };
  let ws = null;
  let reconnectTimer = null;
  let backoffMs = 1000;

  const metricDefs = [
    ["speed_kmh", "Скорость", "км/ч"],
    ["fuel_level_l", "Топливо", "л"],
    ["fuel_rate_lph", "Расход", "л/ч"],
    ["brake_pipe_pressure_bar", "Торм. магистраль", "бар"],
    ["main_reservoir_bar", "Главный резервуар", "бар"],
    ["engine_oil_pressure_bar", "Давл. масла", "бар"],
    ["coolant_temp_c", "ОЖ", "°C"],
    ["engine_oil_temp_c", "Масло двиг.", "°C"],
    ["battery_voltage_v", "АКБ", "В"],
    ["traction_current_a", "Ток тяги", "А"],
    ["line_voltage_v", "Контактная сеть", "В"],
    ["mileage_km", "Пробег (уч.)", "км"],
  ];

  function trainId() {
    return (trainInput.value || "LOC-DEMO-001").trim();
  }

  function setConnected(ok, text) {
    connPill.textContent = text;
    connPill.className = "pill " + (ok ? "ok" : "bad");
  }

  function categoryLabel(hi) {
    if (hi >= 85) return "Норма";
    if (hi >= 60) return "Внимание";
    return "Критично";
  }

  function levelFromHI(hi) {
    if (hi >= 85) return "norm";
    if (hi >= 60) return "attention";
    return "critical";
  }

  function fmt(v, dec) {
    if (v == null || Number.isNaN(v)) return "—";
    if (typeof v === "number") return dec != null ? v.toFixed(dec) : String(v);
    return String(v);
  }

  function renderSample(s) {
    const hi = typeof s.health_index === "number" ? s.health_index : 0;
    healthValue.textContent = fmt(hi, 0);
    healthGrade.textContent = s.health_grade || "—";
    healthRing.dataset.level = levelFromHI(hi);
    healthRing.style.setProperty("--pct", String(Math.max(0, Math.min(100, hi))));
    healthCategory.textContent = categoryLabel(hi) + " · индекс " + fmt(hi, 1);

    factorsEl.innerHTML = "";
    const tops = s.health_top_factors || [];
    if (tops.length === 0) {
      factorsEl.innerHTML = '<p class="muted small">Нет активных штрафов</p>';
    } else {
      tops.forEach((f) => {
        const row = document.createElement("div");
        row.className = "factor-row";
        row.innerHTML =
          '<span>' +
          escapeHtml(f.factor) +
          '</span><span class="muted">−' +
          fmt(f.penalty, 1) +
          "</span>";
        factorsEl.appendChild(row);
      });
    }

    metricsEl.innerHTML = "";
    metricDefs.forEach(([key, label, unit]) => {
      let val = s[key];
      if (key === "traction_current_a") val = s.traction_current_a;
      const m = document.createElement("div");
      m.className = "metric";
      m.innerHTML =
        '<div class="k">' +
        escapeHtml(label) +
        '</div><div class="v">' +
        escapeHtml(fmtVal(key, val)) +
        " " +
        escapeHtml(unit) +
        "</div>";
      metricsEl.appendChild(m);
    });

    motorsEl.innerHTML = "";
    (s.traction_motor_temp_c || []).forEach((t, i) => {
      const div = document.createElement("div");
      div.className = "motor";
      if (t > 115) div.classList.add("crit");
      else if (t > 105) div.classList.add("hot");
      div.innerHTML =
        '<div class="i">ТЭД ' +
        (i + 1) +
        '</div><div class="t">' +
        fmt(t, 1) +
        "°</div>";
      motorsEl.appendChild(div);
    });

    alertsEl.innerHTML = "";
    const alerts = s.alerts || [];
    if (alerts.length === 0) {
      alertsEl.innerHTML = '<li class="muted">нет активных</li>';
    } else {
      alerts.forEach((a) => {
        const li = document.createElement("li");
        const sev = (a.severity || "info").toLowerCase();
        li.className = sev === "crit" ? "crit" : sev === "warn" ? "warn" : "info";
        li.textContent = (a.code ? "[" + a.code + "] " : "") + (a.text || "");
        alertsEl.appendChild(li);
      });
    }

    lastTs.textContent =
      "последнее обновление: " + (s.ts || new Date().toISOString());

    updateMap(s);
  }

  function applySample(s) {
    renderSample(s);
    pushBuffers(s);
    updateCharts();
  }

  function fmtVal(key, val) {
    if (typeof val === "number") {
      if (key.includes("pressure") || key.includes("temp")) return val.toFixed(1);
      if (key === "mileage_km") return val.toFixed(2);
      if (key === "fuel_level_l") return val.toFixed(0);
      return Number.isInteger(val) ? val : val.toFixed(1);
    }
    return val;
  }

  function escapeHtml(s) {
    const d = document.createElement("div");
    d.textContent = s;
    return d.innerHTML;
  }

  let latMin = 51.1,
    latMax = 51.2,
    lonMin = 71.4,
    lonMax = 71.5;

  function updateMap(s) {
    const lat = s.lat,
      lon = s.lon;
    if (typeof lat === "number" && typeof lon === "number") {
      latMin = Math.min(latMin, lat - 0.02);
      latMax = Math.max(latMax, lat + 0.02);
      lonMin = Math.min(lonMin, lon - 0.02);
      lonMax = Math.max(lonMax, lon + 0.02);
      const tLon = (lon - lonMin) / (lonMax - lonMin || 1);
      const cx = 32 + Math.max(0, Math.min(1, tLon)) * 336;
      mapDot.setAttribute("cx", cx.toFixed(1));
      mapCaption.textContent =
        lat.toFixed(4) + "°, " + lon.toFixed(4) + "° · пробег " + fmt(s.mileage_km, 2) + " км";
    }
  }

  function pushBuffers(s) {
    const label = new Date(s.ts || Date.now()).toLocaleTimeString();
    buffers.labels.push(label);
    buffers.speed.push(s.speed_kmh ?? null);
    buffers.coolant.push(s.coolant_temp_c ?? null);
    buffers.health.push(s.health_index ?? null);
    while (buffers.labels.length > MAX_POINTS) {
      buffers.labels.shift();
      buffers.speed.shift();
      buffers.coolant.shift();
      buffers.health.shift();
    }
  }

  function chartOpts(title) {
    const grid = getComputedStyle(document.documentElement).getPropertyValue("--border").trim();
    const text = getComputedStyle(document.documentElement).getPropertyValue("--muted").trim();
    return {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        title: { display: true, text: title, color: text || "#888", font: { size: 11 } },
      },
      scales: {
        x: {
          display: buffers.labels.length > 2,
          ticks: { maxTicksLimit: 8, color: text },
          grid: { color: grid },
        },
        y: {
          beginAtZero: false,
          ticks: { color: text },
          grid: { color: grid },
        },
      },
    };
  }

  function initCharts() {
    const accent = "#58a6ff";
    const ok = "#3fb950";
    const warn = "#f0883e";

    charts.speed = new Chart(document.getElementById("chartSpeed"), {
      type: "line",
      data: {
        labels: [],
        datasets: [{ label: "км/ч", data: [], borderColor: accent, tension: 0.2, fill: false }],
      },
      options: chartOpts("Скорость"),
    });
    charts.temp = new Chart(document.getElementById("chartTemp"), {
      type: "line",
      data: {
        labels: [],
        datasets: [{ label: "ОЖ °C", data: [], borderColor: warn, tension: 0.2, fill: false }],
      },
      options: chartOpts("Температура ОЖ"),
    });
    charts.health = new Chart(document.getElementById("chartHealth"), {
      type: "line",
      data: {
        labels: [],
        datasets: [{ label: "индекс", data: [], borderColor: ok, tension: 0.2, fill: false }],
      },
      options: chartOpts("Индекс здоровья"),
    });
  }

  function updateCharts() {
    if (!charts.speed) return;
    ["speed", "temp", "health"].forEach((k, i) => {
      const ds =
        i === 0 ? buffers.speed : i === 1 ? buffers.coolant : buffers.health;
      const ch = charts[k];
      ch.data.labels = buffers.labels.slice();
      ch.data.datasets[0].data = ds.slice();
      ch.update("none");
    });
  }

  function connectWS() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const url = proto + "//" + location.host + "/ws/telemetry";
    setConnected(false, "подключение…");
    ws = new WebSocket(url);
    ws.onopen = function () {
      backoffMs = 1000;
      setConnected(true, "онлайн");
    };
    ws.onclose = function () {
      setConnected(false, "нет связи · переподключение…");
      reconnectTimer = setTimeout(connectWS, backoffMs);
      backoffMs = Math.min(backoffMs * 2, 30000);
    };
    ws.onerror = function () {
      try {
        ws.close();
      } catch (_) {}
    };
    ws.onmessage = function (ev) {
      try {
        const s = JSON.parse(ev.data);
        if (s.train_id && s.train_id !== trainId()) return;
        applySample(s);
      } catch (_) {}
    };
  }

  async function loadHistory() {
    const tid = encodeURIComponent(trainId());
    const res = await fetch("/api/v1/telemetry/history?train_id=" + tid + "&limit=300");
    if (!res.ok) return;
    const list = await res.json();
    if (!Array.isArray(list) || list.length === 0) {
      const r2 = await fetch("/api/v1/telemetry/latest?train_id=" + tid);
      if (r2.ok) {
        const s = await r2.json();
        applySample(s);
      }
      return;
    }
    const asc = list.slice().reverse();
    buffers.labels = [];
    buffers.speed = [];
    buffers.coolant = [];
    buffers.health = [];
    asc.forEach(function (s) {
      pushBuffers(s);
    });
    renderSample(asc[asc.length - 1]);
    updateCharts();
  }

  trainInput.addEventListener("change", function () {
    buffers.labels = [];
    buffers.speed = [];
    buffers.coolant = [];
    buffers.health = [];
    loadHistory().catch(function () {});
  });

  btnTheme.addEventListener("click", function () {
    const app = document.querySelector(".app");
    const t = app.getAttribute("data-theme") === "light" ? "dark" : "light";
    app.setAttribute("data-theme", t);
  });

  document.addEventListener("DOMContentLoaded", function () {
    if (typeof Chart !== "undefined") {
      try {
        initCharts();
      } catch (e) {
        console.warn("Chart.js init:", e);
      }
    } else {
      console.warn(
        "Chart.js не загрузился (проверьте интернет / CDN). Графики отключены; метрики и WebSocket работают."
      );
    }
    loadHistory()
      .catch(function () {})
      .then(function () {
        connectWS();
      });
  });
})();
