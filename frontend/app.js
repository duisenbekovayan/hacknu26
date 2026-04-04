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
  const mapSubcaption = document.getElementById("mapSubcaption");
  const btnTheme = document.getElementById("btnTheme");
  const aiStatusLine = document.getElementById("aiStatusLine");
  const btnAIExplain = document.getElementById("btnAIExplain");
  const btnAIActions = document.getElementById("btnAIActions");
  const aiPanel = document.getElementById("aiPanel");
  const aiSeverity = document.getElementById("aiSeverity");
  const aiSummary = document.getElementById("aiSummary");
  const aiCauses = document.getElementById("aiCauses");
  const aiRecs = document.getElementById("aiRecs");
  const aiMetrics = document.getElementById("aiMetrics");
  const aiRisk = document.getElementById("aiRisk");
  const aiErr = document.getElementById("aiErr");

  let lastSample = null;
  let aiEnabled = false;

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
  let pollTimer = null;
  /** последний применённый ts (дедуп WS + polling) */
  let lastAppliedTs = "";

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

  function sameTrain(msgId, selectedId) {
    const a = (msgId == null ? "" : String(msgId)).trim();
    const b = (selectedId == null ? "" : String(selectedId)).trim();
    if (!a) return true;
    return a === b;
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
    if (s && s.ts) lastAppliedTs = s.ts;
    lastSample = s;
    renderSample(s);
    pushBuffers(s);
    try {
      updateCharts();
    } catch (e) {
      console.warn("chart update", e);
    }
  }

  function msgTrainId(s) {
    if (s == null) return "";
    var id = s.train_id != null ? s.train_id : s.TrainID;
    return id;
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

  /** Условный участок: населённые пункты по км, зоны Vmax (как на схемах перегонов) */
  const ROUTE = {
    loopKm: 42,
    settlements: [
      { km: 0, name: "Астана", major: true },
      { km: 10, name: "Жетыген" },
      { km: 18, name: "Мойынты" },
      { km: 22, name: "Балхаш" },
      { km: 32, name: "Караганда", major: true },
      { km: 38, name: "Темиртау" },
    ],
    segments: [
      { from: 0, to: 10, vMax: 70, restriction: "" },
      { from: 10, to: 18, vMax: 90, restriction: "" },
      { from: 18, to: 22, vMax: 40, restriction: "Контактная сеть · работы" },
      { from: 22, to: 32, vMax: 90, restriction: "" },
      { from: 32, to: 38, vMax: 60, restriction: "" },
      { from: 38, to: 42, vMax: 50, restriction: "Стрелки · осторожно" },
    ],
    /** Упрощённая привязка карты к км участка (интерполяция; совпадает с движением кружка на схеме). */
    geoPath: [
      { km: 0, lat: 51.1694, lng: 71.4491 },
      { km: 10, lat: 51.158, lng: 71.468 },
      { km: 18, lat: 51.152, lng: 71.492 },
      { km: 22, lat: 51.148, lng: 71.505 },
      { km: 32, lat: 51.135, lng: 71.528 },
      { km: 38, lat: 51.128, lng: 71.545 },
      { km: 42, lat: 51.1694, lng: 71.4491 },
    ],
  };

  /** Горизонталь пути в единицах viewBox: линейный масштаб 1 км → фиксированная доля длины. */
  const MAP_X0 = 44;
  const MAP_X1 = 476;
  const RAIL_Y = 58;
  const RAIL_Y1 = 53;
  const RAIL_Y2 = 63;

  function kmToPx(km) {
    const t = km / ROUTE.loopKm;
    return MAP_X0 + Math.max(0, Math.min(1, t)) * (MAP_X1 - MAP_X0);
  }

  function zoneClass(vMax) {
    if (vMax >= 85) return "map-zone map-zone--xfast";
    if (vMax >= 60) return "map-zone map-zone--fast";
    if (vMax >= 45) return "map-zone map-zone--med";
    return "map-zone map-zone--slow";
  }

  function findSegment(km) {
    const x = ((km % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    for (let i = 0; i < ROUTE.segments.length; i++) {
      const s = ROUTE.segments[i];
      if (x >= s.from && x < s.to) {
        return s;
      }
    }
    return ROUTE.segments[ROUTE.segments.length - 1];
  }

  /** Координаты на карте по положению на участке (согласовано с mileage_km и синим маркером). */
  function geoAtKm(km) {
    const P = ROUTE.geoPath;
    const x = ((km % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    for (let i = 0; i < P.length - 1; i++) {
      const a = P[i];
      const b = P[i + 1];
      if (x >= a.km && x <= b.km) {
        const span = b.km - a.km;
        const t = span <= 0 ? 0 : (x - a.km) / span;
        return {
          lat: a.lat + t * (b.lat - a.lat),
          lng: a.lng + t * (b.lng - a.lng),
        };
      }
    }
    return { lat: P[0].lat, lng: P[0].lng };
  }

  /** Между какими пунктами сейчас состав (для подписи) */
  function routeLegAt(km) {
    const S = ROUTE.settlements;
    const x = ((km % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    for (let i = 0; i < S.length - 1; i++) {
      if (x >= S[i].km && x < S[i + 1].km) {
        return {
          from: S[i],
          to: S[i + 1],
          legKm: S[i + 1].km - S[i].km,
          wrap: false,
        };
      }
    }
    return {
      from: S[S.length - 1],
      to: S[0],
      legKm: ROUTE.loopKm - S[S.length - 1].km,
      wrap: true,
    };
  }

  function initRouteScheme() {
    const zones = document.getElementById("routeZones");
    const railG = document.getElementById("routeRail");
    const segKmG = document.getElementById("routeSegKm");
    const restrG = document.getElementById("routeRestrictions");
    const stations = document.getElementById("routeStations");
    const leg = document.getElementById("mapLegend");
    if (!zones || !railG || !segKmG || !restrG || !stations) return;

    zones.textContent = "";
    ROUTE.segments.forEach(function (seg) {
      const r = document.createElementNS("http://www.w3.org/2000/svg", "rect");
      const x0 = kmToPx(seg.from);
      const x1 = kmToPx(seg.to);
      r.setAttribute("x", String(x0));
      r.setAttribute("y", "40");
      r.setAttribute("width", String(Math.max(0.5, x1 - x0)));
      r.setAttribute("height", "16");
      r.setAttribute("rx", "2");
      r.setAttribute("class", zoneClass(seg.vMax));
      zones.appendChild(r);
    });

    railG.textContent = "";
    const railTop = document.createElementNS("http://www.w3.org/2000/svg", "line");
    railTop.setAttribute("x1", String(MAP_X0));
    railTop.setAttribute("y1", String(RAIL_Y1));
    railTop.setAttribute("x2", String(MAP_X1));
    railTop.setAttribute("y2", String(RAIL_Y1));
    railTop.setAttribute("class", "map-rail-line");
    railG.appendChild(railTop);
    const railBot = document.createElementNS("http://www.w3.org/2000/svg", "line");
    railBot.setAttribute("x1", String(MAP_X0));
    railBot.setAttribute("y1", String(RAIL_Y2));
    railBot.setAttribute("x2", String(MAP_X1));
    railBot.setAttribute("y2", String(RAIL_Y2));
    railBot.setAttribute("class", "map-rail-line");
    railG.appendChild(railBot);
    for (let km = 0; km <= ROUTE.loopKm; km += 1) {
      const rx = kmToPx(km);
      const sl = document.createElementNS("http://www.w3.org/2000/svg", "line");
      sl.setAttribute("x1", String(rx));
      sl.setAttribute("x2", String(rx));
      sl.setAttribute("y1", String(RAIL_Y1 - 1));
      sl.setAttribute("y2", String(RAIL_Y2 + 1));
      sl.setAttribute("class", km % 10 === 0 ? "map-sleeper map-sleeper--10" : "map-sleeper");
      railG.appendChild(sl);
    }

    segKmG.textContent = "";
    const S = ROUTE.settlements;
    function addSegKmLabel(kmA, kmB) {
      const mid = (kmA + kmB) / 2;
      const dkm = kmB - kmA;
      if (dkm <= 0) return;
      const tx = document.createElementNS("http://www.w3.org/2000/svg", "text");
      tx.setAttribute("x", String(kmToPx(mid)));
      tx.setAttribute("y", "32");
      tx.setAttribute("text-anchor", "middle");
      tx.setAttribute("class", "map-seg-km");
      tx.textContent = dkm + " км";
      segKmG.appendChild(tx);
    }
    for (let i = 0; i < S.length - 1; i++) {
      addSegKmLabel(S[i].km, S[i + 1].km);
    }
    if (S[S.length - 1].km < ROUTE.loopKm) {
      addSegKmLabel(S[S.length - 1].km, ROUTE.loopKm);
    }

    restrG.textContent = "";
    ROUTE.segments.forEach(function (seg) {
      if (!seg.restriction) return;
      const cx = (kmToPx(seg.from) + kmToPx(seg.to)) / 2;
      const t = document.createElementNS("http://www.w3.org/2000/svg", "text");
      t.setAttribute("x", String(cx));
      t.setAttribute("y", "118");
      t.setAttribute("text-anchor", "middle");
      t.setAttribute("class", "map-restr");
      t.textContent = seg.restriction;
      restrG.appendChild(t);
    });

    stations.textContent = "";
    ROUTE.settlements.forEach(function (st) {
      const x = kmToPx(st.km);
      const nc = document.createElementNS("http://www.w3.org/2000/svg", "circle");
      nc.setAttribute("cx", String(x));
      nc.setAttribute("cy", String(RAIL_Y));
      nc.setAttribute("r", st.major ? "6" : "4.5");
      nc.setAttribute("class", st.major ? "map-node map-node--major" : "map-node");
      nc.setAttribute("title", st.name + " · " + st.km + " км");
      stations.appendChild(nc);
      const lab = document.createElementNS("http://www.w3.org/2000/svg", "text");
      lab.setAttribute("x", String(x));
      lab.setAttribute("y", "82");
      lab.setAttribute("text-anchor", "middle");
      lab.setAttribute("class", "map-np-name");
      lab.textContent = st.name;
      stations.appendChild(lab);
      const kmL = document.createElementNS("http://www.w3.org/2000/svg", "text");
      kmL.setAttribute("x", String(x));
      kmL.setAttribute("y", "94");
      kmL.setAttribute("text-anchor", "middle");
      kmL.setAttribute("class", "map-np-km");
      kmL.textContent = st.km + " км";
      stations.appendChild(kmL);
    });

    if (leg) {
      leg.innerHTML =
        '<span><i style="background:rgba(63,185,80,0.35)"></i> 85+ км/ч</span>' +
        '<span><i style="background:rgba(88,166,255,0.35)"></i> 60–84</span>' +
        '<span><i style="background:rgba(210,153,34,0.35)"></i> 45–59</span>' +
        '<span><i style="background:rgba(248,81,73,0.28)"></i> &lt;45</span>' +
        '<span class="map-legend-note">· узлы — НП, красные цифры — длина перегона</span>';
    }
  }

  function updateMap(s) {
    const mileage = typeof s.mileage_km === "number" ? s.mileage_km : 0;
    const km = ((mileage % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    const cx = kmToPx(km);
    mapDot.setAttribute("cx", cx.toFixed(1));
    mapDot.setAttribute("cy", String(RAIL_Y));

    const seg = findSegment(km);
    const vMax = seg.vMax;
    const spd = typeof s.speed_kmh === "number" ? s.speed_kmh : 0;
    const over = spd > vMax + 2;

    mapDot.classList.toggle("map-dot--warn", over);
    mapCaption.classList.toggle("map-caption--warn", over);

    const g = geoAtKm(km);
    const geo = g.lat.toFixed(5) + "°, " + g.lng.toFixed(5) + "° · ";
    mapCaption.textContent =
      geo +
      "путь " +
      km.toFixed(2) +
      " км (цикл " +
      ROUTE.loopKm +
      " км) · Vmax " +
      vMax +
      " · ход " +
      spd.toFixed(0) +
      " км/ч" +
      (over ? " — превышение!" : "");

    if (mapSubcaption) {
      const leg = routeLegAt(km);
      let sub = leg.wrap
        ? "между " + leg.from.name + " и " + leg.to.name + " (замыкание кольца) · " + leg.legKm + " км"
        : "между " + leg.from.name + " и " + leg.to.name + " · " + leg.legKm + " км";
      sub += " · Vmax " + vMax + " км/ч";
      if (seg.restriction) sub += " · " + seg.restriction;
      mapSubcaption.textContent = sub;
    }

    updateGmapFromSample({ lat: g.lat, lon: g.lng });
  }

  /** Опционально: Google Maps по lat/lon из телеметрии (ключ с сервера). */
  var gmapApi = { map: null, marker: null, ready: false, pending: null, scriptRequested: false };

  var GMAP_DARK_STYLES = [
    { elementType: "geometry", stylers: [{ color: "#1d2330" }] },
    { elementType: "labels.text.fill", stylers: [{ color: "#8b949e" }] },
    { featureType: "road", elementType: "geometry", stylers: [{ color: "#30363d" }] },
    { featureType: "water", elementType: "geometry", stylers: [{ color: "#0d1117" }] },
  ];

  function syncGmapTheme() {
    if (!gmapApi.map || !window.google || !google.maps) return;
    var dark =
      document.querySelector(".app") &&
      document.querySelector(".app").getAttribute("data-theme") === "dark";
    gmapApi.map.setOptions({ styles: dark ? GMAP_DARK_STYLES : null });
  }

  window.hacknuGoogleMapReady = function () {
    var el = document.getElementById("gmap");
    if (!el || !window.google || !google.maps) return;
    var center = { lat: 51.15, lng: 71.43 };
    gmapApi.map = new google.maps.Map(el, {
      zoom: 10,
      center: center,
      mapTypeControl: false,
      streetViewControl: false,
      fullscreenControl: true,
    });
    gmapApi.marker = new google.maps.Marker({
      position: center,
      map: gmapApi.map,
      title: "Положение состава",
    });
    gmapApi.ready = true;
    syncGmapTheme();
    if (gmapApi.pending) {
      gmapApi.marker.setPosition(gmapApi.pending);
      gmapApi.map.panTo(gmapApi.pending);
      gmapApi.pending = null;
    }
    google.maps.event.addListenerOnce(gmapApi.map, "idle", function () {
      google.maps.event.trigger(gmapApi.map, "resize");
    });
  };

  function updateGmapFromSample(s) {
    if (typeof s.lat !== "number" || typeof s.lon !== "number") return;
    var pos = { lat: s.lat, lng: s.lon };
    if (!gmapApi.ready) {
      gmapApi.pending = pos;
      return;
    }
    gmapApi.marker.setPosition(pos);
    gmapApi.map.panTo(pos);
  }

  async function initGoogleMapsFromConfig() {
    var hint = document.getElementById("gmapHint");
    var el = document.getElementById("gmap");
    if (!el) return;
    try {
      var res = await fetch("/api/v1/config", fetchOpts());
      if (!res.ok) throw new Error("config");
      var cfg = await res.json();
      var key = (cfg.google_maps_api_key || "").trim();
      if (!key) {
        if (hint) {
          hint.textContent =
            "Карта: задайте GOOGLE_MAPS_API_KEY (Maps JavaScript API) на сервере — см. README.";
        }
        return;
      }
      if (hint) hint.textContent = "Загрузка карты…";
      if (gmapApi.scriptRequested) return;
      gmapApi.scriptRequested = true;
      el.hidden = false;
      var scr = document.createElement("script");
      scr.async = true;
      scr.defer = true;
      scr.setAttribute("data-hacknu-gmaps", "1");
      scr.src =
        "https://maps.googleapis.com/maps/api/js?key=" +
        encodeURIComponent(key) +
        "&callback=hacknuGoogleMapReady";
      scr.onerror = function () {
        if (hint) {
          hint.textContent =
            "Не удалось загрузить Google Maps (ключ, биллинг или ограничения API).";
        }
        gmapApi.scriptRequested = false;
      };
      document.head.appendChild(scr);
      if (hint) hint.textContent = "";
    } catch (e) {
      console.warn("gmaps", e);
      if (hint) hint.textContent = "Карта: ошибка запроса /api/v1/config.";
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
    if (ws) {
      try {
        ws.onclose = null;
        ws.close();
      } catch (_) {}
      ws = null;
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
      var p = typeof ev.data === "string" ? Promise.resolve(ev.data) : ev.data instanceof Blob ? ev.data.text() : Promise.resolve(String(ev.data));
      p.then(function (raw) {
        var s = JSON.parse(raw);
        if (!sameTrain(msgTrainId(s), trainId())) return;
        applySample(s);
      }).catch(function (e) {
        console.warn("ws message", e);
      });
    };
  }

  function fetchOpts() {
    return { cache: "no-store", headers: { Pragma: "no-cache" } };
  }

  function pollLatest() {
    if (document.visibilityState !== "visible") return;
    const tid = encodeURIComponent(trainId());
    fetch("/api/v1/telemetry/latest?train_id=" + tid, fetchOpts())
      .then(function (r) {
        if (!r.ok) return null;
        return r.json();
      })
      .then(function (s) {
        if (!s || !s.ts) return;
        if (s.ts === lastAppliedTs) return;
        applySample(s);
      })
      .catch(function () {});
  }

  function startLivePoll() {
    if (pollTimer) clearInterval(pollTimer);
    pollTimer = setInterval(pollLatest, 1000);
    pollLatest();
  }

  async function loadHistory() {
    const tid = encodeURIComponent(trainId());
    const res = await fetch(
      "/api/v1/telemetry/history?train_id=" + tid + "&limit=300",
      fetchOpts()
    );
    if (!res.ok) return;
    const list = await res.json();
    if (!Array.isArray(list) || list.length === 0) {
      const r2 = await fetch("/api/v1/telemetry/latest?train_id=" + tid, fetchOpts());
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
    const last = asc[asc.length - 1];
    lastAppliedTs = last.ts || "";
    renderSample(last);
    try {
      updateCharts();
    } catch (e) {
      console.warn("chart update", e);
    }
  }

  trainInput.addEventListener("change", function () {
    buffers.labels = [];
    buffers.speed = [];
    buffers.coolant = [];
    buffers.health = [];
    loadHistory().catch(function () {});
  });

  function buildTrendNotes() {
    const notes = [];
    const n = buffers.labels.length;
    if (n < 2) return notes;
    const span = Math.min(30, n - 1);
    const i0 = n - 1 - span;
    const sp0 = buffers.speed[i0],
      sp1 = buffers.speed[n - 1];
    const c0 = buffers.coolant[i0],
      c1 = buffers.coolant[n - 1];
    const h0 = buffers.health[i0],
      h1 = buffers.health[n - 1];
    if (sp0 != null && sp1 != null && Math.abs(sp1 - sp0) > 0.3) {
      notes.push("скорость: " + sp0.toFixed(1) + " → " + sp1.toFixed(1) + " км/ч");
    }
    if (c0 != null && c1 != null && Math.abs(c1 - c0) > 0.5) {
      notes.push("ОЖ: " + c0.toFixed(1) + " → " + c1.toFixed(1) + " °C");
    }
    if (h0 != null && h1 != null && Math.abs(h1 - h0) > 1) {
      notes.push("индекс здоровья: " + h0.toFixed(0) + " → " + h1.toFixed(0));
    }
    return notes.slice(0, 5);
  }

  function buildAnalyzePayload(s, mode) {
    const alerts = (s.alerts || []).map(function (a) {
      return (a.code ? "[" + a.code + "] " : "") + (a.text || "");
    });
    const body = {
      train_id: s.train_id || "",
      timestamp: s.ts || "",
      health_index: typeof s.health_index === "number" ? s.health_index : 0,
      speed: typeof s.speed_kmh === "number" ? s.speed_kmh : 0,
      fuel_level: typeof s.fuel_level_l === "number" ? s.fuel_level_l : 0,
      engine_temp: typeof s.coolant_temp_c === "number" ? s.coolant_temp_c : 0,
      brake_pressure: typeof s.brake_pipe_pressure_bar === "number" ? s.brake_pipe_pressure_bar : 0,
      voltage: typeof s.battery_voltage_v === "number" ? s.battery_voltage_v : 0,
      current: typeof s.traction_current_a === "number" ? s.traction_current_a : 0,
      alerts: alerts,
      recent_trend_notes: buildTrendNotes(),
    };
    if (mode) body.mode = mode;
    return body;
  }

  function sevKey(sev) {
    const x = (sev || "").toLowerCase();
    if (x === "critical" || x === "crit") return "critical";
    if (x === "warning" || x === "warn") return "warning";
    return "normal";
  }

  function renderAIOut(data) {
    aiErr.hidden = true;
    aiPanel.hidden = false;
    const sk = sevKey(data.severity);
    aiSeverity.textContent =
      sk === "critical" ? "критично" : sk === "warning" ? "внимание" : "норма";
    aiSeverity.setAttribute("data-sev", sk);
    aiSummary.textContent = data.summary || "—";

    function fillList(ul, items) {
      ul.innerHTML = "";
      (items || []).forEach(function (t) {
        const li = document.createElement("li");
        li.textContent = t;
        ul.appendChild(li);
      });
      if (!items || items.length === 0) {
        const li = document.createElement("li");
        li.className = "muted";
        li.textContent = "—";
        ul.appendChild(li);
      }
    }
    fillList(aiCauses, data.probable_causes);
    fillList(aiRecs, data.recommendations);

    const am = data.affected_metrics || [];
    aiMetrics.textContent =
      am.length > 0 ? "Метрики: " + am.join(", ") : "Метрики: —";

    if (data.next_risk && String(data.next_risk).trim()) {
      aiRisk.hidden = false;
      aiRisk.textContent = "Риск далее: " + data.next_risk;
    } else {
      aiRisk.hidden = true;
      aiRisk.textContent = "";
    }
  }

  function setAILoading(loading) {
    btnAIExplain.disabled = loading;
    btnAIActions.disabled = loading || !aiEnabled;
    if (loading) {
      aiStatusLine.textContent = "запрос к модели…";
    } else {
      aiStatusLine.textContent = aiEnabled
        ? "ИИ готов (on-demand, без каждого тика телеметрии)."
        : "ИИ отключён: задайте GEMINI_API_KEY или OPENAI_API_KEY на сервере.";
    }
  }

  async function runAI(mode) {
    if (!aiEnabled) {
      aiErr.textContent = "Сервер без GEMINI_API_KEY / OPENAI_API_KEY — см. README.";
      aiErr.hidden = false;
      return;
    }
    if (!lastSample || !lastSample.ts) {
      aiErr.textContent = "Нет данных телеметрии для выбранного поезда.";
      aiErr.hidden = false;
      return;
    }
    aiErr.hidden = true;
    setAILoading(true);
    try {
      const res = await fetch("/api/v1/ai/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(buildAnalyzePayload(lastSample, mode)),
      });
      if (!res.ok) {
        const t = await res.text();
        let msg = t;
        try {
          const j = JSON.parse(t);
          if (j && typeof j.error === "string" && j.error) msg = j.error;
        } catch (_) {
          msg = (t || "").trim() || res.statusText;
        }
        throw new Error(msg);
      }
      const data = await res.json();
      renderAIOut(data);
    } catch (e) {
      aiErr.textContent = "Ошибка: " + (e && e.message ? e.message : String(e));
      aiErr.hidden = false;
    } finally {
      setAILoading(false);
    }
  }

  async function refreshAIStatus() {
    try {
      const res = await fetch("/api/v1/ai/status", fetchOpts());
      if (!res.ok) throw new Error("status");
      const j = await res.json();
      aiEnabled = !!j.enabled;
      aiStatusLine.textContent = aiEnabled
        ? "ИИ готов (on-demand, без каждого тика телеметрии)."
        : "ИИ отключён: задайте GEMINI_API_KEY или OPENAI_API_KEY на сервере.";
      btnAIExplain.disabled = false;
      btnAIActions.disabled = !aiEnabled;
    } catch (_) {
      aiEnabled = false;
      aiStatusLine.textContent = "Не удалось проверить /api/v1/ai/status";
      btnAIActions.disabled = true;
    }
  }

  btnAIExplain.addEventListener("click", function () {
    runAI("");
  });
  btnAIActions.addEventListener("click", function () {
    runAI("actions");
  });

  btnTheme.addEventListener("click", function () {
    const app = document.querySelector(".app");
    const t = app.getAttribute("data-theme") === "light" ? "dark" : "light";
    app.setAttribute("data-theme", t);
    syncGmapTheme();
  });

  document.addEventListener("DOMContentLoaded", function () {
    initGoogleMapsFromConfig().catch(function () {});
    try {
      initRouteScheme();
    } catch (e) {
      console.warn("route scheme", e);
    }
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
        refreshAIStatus().catch(function () {});
        connectWS();
        startLivePoll();
      });
  });
})();
