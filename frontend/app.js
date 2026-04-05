/* global Chart, ChartZoom */
(function () {
  const MAX_POINTS = 2000;
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
  const recommendationsEl = document.getElementById("recommendations");
  const lastTs = document.getElementById("lastTs");
  const mapDot = document.getElementById("mapDot");
  const mapCaption = document.getElementById("mapCaption");
  const mapSubcaption = document.getElementById("mapSubcaption");
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
  const btnTheme = document.getElementById("btnTheme");
  const replayMinutesEl = document.getElementById("replayMinutes");
  const btnLoadReplay = document.getElementById("btnLoadReplay");
  const replayScrubWrap = document.getElementById("replayScrubWrap");
  const replaySlider = document.getElementById("replaySlider");
  const replayTimeLabel = document.getElementById("replayTimeLabel");
  const btnReplayPlay = document.getElementById("btnReplayPlay");

  const buffers = {
    labels: [],
    speed: [],
    coolant: [],
    health: [],
    fuel: [],
  };

  let charts = { speed: null, temp: null, health: null, fuel: null };
  let ws = null;
  let reconnectTimer = null;
  let backoffMs = 1000;
  let pollTimer = null;
  let lastAppliedTs = "";
  let replayMode = false;
  let replaySamples = [];
  let replayIndex = 0;
  let playTimer = null;
  let playing = false;
  let lastSample = null;
  let aiEnabled = false;

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

  const factorActions = [
    { re: /перегрев ОЖ/i, text: "Снизить нагрузку двигателя; проверить охлаждение и уровень ОЖ." },
    { re: /низкое давление масла/i, text: "Не наращивать тягу; проверить уровень и давление масла по регламенту." },
    { re: /напряжение АКБ/i, text: "Проверить заряд АКБ и цепи генератора/выпрямителя." },
    { re: /ТЭД\d+/i, text: "Проверить вентиляцию и нагрузку на тяговые двигатели; избегать длительной работы на пределе." },
    { re: /низкое давление ГР/i, text: "Проверить питание сжатым воздухом и тормозную магистраль." },
    { re: /^алерт\s+/i, text: "Свериться с кодом алерта в инструкции; при необходимости снизить ход." },
  ];

  function trainId() {
    return (trainInput.value || "LOC-DEMO-001").trim();
  }

  function windowMinutes() {
    const v = parseInt(replayMinutesEl && replayMinutesEl.value, 10);
    if (v === 5 || v === 10 || v === 15) return v;
    return 15;
  }

  function sameTrain(msgId, selectedId) {
    const a = (msgId == null ? "" : String(msgId)).trim();
    const b = (selectedId == null ? "" : String(selectedId)).trim();
    if (!a) return true;
    return a.toLowerCase() === b.toLowerCase();
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

  function recommendationsFor(s) {
    const out = [];
    const hi = typeof s.health_index === "number" ? s.health_index : 0;
    (s.health_top_factors || []).forEach(function (f) {
      const name = f.factor || "";
      for (let i = 0; i < factorActions.length; i++) {
        if (factorActions[i].re.test(name)) {
          out.push(factorActions[i].text);
          return;
        }
      }
    });
    (s.alerts || []).forEach(function (a) {
      const sev = (a.severity || "").toLowerCase();
      const code = a.code ? String(a.code) : "";
      if (sev === "crit") {
        out.push("Критический сигнал " + code + ": быть готовым к остановке или снижению хода по регламенту.");
      } else if (sev === "warn") {
        out.push("Предупреждение " + code + ": проверить узел до усугубления; зафиксировать событие.");
      }
    });
    if (out.length === 0) {
      if (hi >= 85) {
        out.push("Состояние в норме; продолжайте мониторинг ключевых параметров.");
      } else {
        out.push("Удерживайте параметры из списка факторов под контролем; при падении индекса — снижать нагрузку.");
      }
    }
    return [...new Set(out)];
  }

  function renderRecommendations(s) {
    recommendationsEl.innerHTML = "";
    const items = recommendationsFor(s);
    items.forEach(function (text) {
      const li = document.createElement("li");
      li.textContent = text;
      recommendationsEl.appendChild(li);
    });
  }

  function renderDashboard(s) {
    lastSample = s;
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
      tops.forEach(function (f) {
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
    metricDefs.forEach(function (def) {
      const key = def[0];
      const label = def[1];
      const unit = def[2];
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
    (s.traction_motor_temp_c || []).forEach(function (t, i) {
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
      alerts.forEach(function (a) {
        const li = document.createElement("li");
        const sev = (a.severity || "info").toLowerCase();
        li.className = sev === "crit" ? "crit" : sev === "warn" ? "warn" : "info";
        li.textContent = (a.code ? "[" + a.code + "] " : "") + (a.text || "");
        alertsEl.appendChild(li);
      });
    }

    renderRecommendations(s);

    if (replayMode) {
      lastTs.textContent =
        "replay: " + (s.ts || "—") + " · кадр " + (replayIndex + 1) + " / " + replaySamples.length;
    } else {
      lastTs.textContent = "последнее обновление: " + (s.ts || new Date().toISOString());
    }

    updateMap(s);
  }

  function applySample(s) {
    if (s && s.ts) lastAppliedTs = s.ts;
    renderDashboard(s);
    pushBuffers(s);
    try {
      updateCharts();
    } catch (e) {
      console.warn("chart update", e);
    }
  }

  function applyReplayFrame(i) {
    if (!replaySamples.length) return;
    replayIndex = Math.max(0, Math.min(i, replaySamples.length - 1));
    const s = replaySamples[replayIndex];
    renderDashboard(s);
    replaySlider.value = String(replayIndex);
    replayTimeLabel.textContent =
      (s.ts || "") + " · " + (replayIndex + 1) + "/" + replaySamples.length;
  }

  function msgTrainId(s) {
    if (s == null) return "";
    const id = s.train_id != null ? s.train_id : s.TrainID;
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

  /**
   * Астана-1 — Шымкент: км вдоль линии пропорционально «времени в пути» из расписания.
   * Полная длина ~1500 км — ориентир по протяжённости главного хода (реальная длина по путям может отличаться).
   */
  const ROUTE = {
    loopKm: 1500,
    settlements: [
      { km: 0, name: "Астана-1", major: true },
      { km: 200, name: "Караганды-Сорт." },
      { km: 245, name: "Караганды-Пасс." },
      { km: 369, name: "Жарык" },
      { km: 433, name: "Акадыр" },
      { km: 575, name: "Мойынты" },
      { km: 673, name: "Сары-Шаган" },
      { km: 801, name: "Шыганак" },
      { km: 995, name: "Шу" },
      { km: 1115, name: "Турксиб", title: "Турксиб (бывш. Луговая)" },
      { km: 1230, name: "Тараз" },
      { km: 1342, name: "Боранды" },
      { km: 1395, name: "Тюлькубас" },
      { km: 1462, name: "Манкент" },
      { km: 1500, name: "Шымкент", major: true },
    ],
    segments: [
      { from: 0, to: 200, vMax: 90, restriction: "" },
      { from: 200, to: 245, vMax: 85, restriction: "" },
      { from: 245, to: 369, vMax: 90, restriction: "" },
      { from: 369, to: 433, vMax: 90, restriction: "" },
      { from: 433, to: 575, vMax: 95, restriction: "" },
      { from: 575, to: 673, vMax: 90, restriction: "" },
      { from: 673, to: 801, vMax: 90, restriction: "" },
      { from: 801, to: 995, vMax: 95, restriction: "Длинный перегон" },
      { from: 995, to: 1115, vMax: 90, restriction: "" },
      { from: 1115, to: 1230, vMax: 90, restriction: "" },
      { from: 1230, to: 1342, vMax: 85, restriction: "" },
      { from: 1342, to: 1395, vMax: 80, restriction: "" },
      { from: 1395, to: 1462, vMax: 85, restriction: "" },
      { from: 1462, to: 1500, vMax: 70, restriction: "Подход к Шымкенту" },
    ],
  };

  /** Координаты в user space SVG (viewBox); на экране масштабируются равномерно — без горизонтального «раздутия» в px */
  const MAP_X0 = 120;
  const MAP_X1 = 8080;
  const RAIL_Y = 110;
  const RAIL_Y1 = 104;
  const RAIL_Y2 = 116;

  const GEO_ASTANA = { lat: 51.1694, lng: 71.4491 };
  const GEO_SHYMKENT = { lat: 42.315, lng: 69.595 };

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
      const seg = ROUTE.segments[i];
      if (x >= seg.from && x < seg.to) return seg;
    }
    return ROUTE.segments[ROUTE.segments.length - 1];
  }

  function routeLegAt(km) {
    const S = ROUTE.settlements;
    const x = ((km % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    for (let i = 0; i < S.length - 1; i++) {
      if (x >= S[i].km && x < S[i + 1].km) {
        return { from: S[i], to: S[i + 1], legKm: S[i + 1].km - S[i].km, wrap: false };
      }
    }
    return {
      from: S[S.length - 1],
      to: S[0],
      legKm: ROUTE.loopKm - S[S.length - 1].km,
      wrap: true,
    };
  }

  function geoAtKm(km) {
    const L = ROUTE.loopKm;
    const x = ((km % L) + L) % L;
    const t = L <= 0 ? 0 : x / L;
    return {
      lat: GEO_ASTANA.lat + t * (GEO_SHYMKENT.lat - GEO_ASTANA.lat),
      lng: GEO_ASTANA.lng + t * (GEO_SHYMKENT.lng - GEO_ASTANA.lng),
    };
  }

  function zoneClassV(vMax) {
    if (vMax >= 85) return "map-zone-pill--xfast";
    if (vMax >= 60) return "map-zone-pill--fast";
    if (vMax >= 45) return "map-zone-pill--med";
    return "map-zone-pill--slow";
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
      r.setAttribute("y", "86");
      r.setAttribute("width", String(Math.max(0.5, x1 - x0)));
      r.setAttribute("height", "48");
      r.setAttribute("class", zoneClass(seg.vMax));
      zones.appendChild(r);
    });

    railG.textContent = "";
    const line1 = document.createElementNS("http://www.w3.org/2000/svg", "line");
    line1.setAttribute("x1", String(MAP_X0));
    line1.setAttribute("y1", String(RAIL_Y1));
    line1.setAttribute("x2", String(MAP_X1));
    line1.setAttribute("y2", String(RAIL_Y1));
    line1.setAttribute("class", "map-rail-line");
    railG.appendChild(line1);
    const line2 = document.createElementNS("http://www.w3.org/2000/svg", "line");
    line2.setAttribute("x1", String(MAP_X0));
    line2.setAttribute("y1", String(RAIL_Y2));
    line2.setAttribute("x2", String(MAP_X1));
    line2.setAttribute("y2", String(RAIL_Y2));
    line2.setAttribute("class", "map-rail-line");
    railG.appendChild(line2);

    var sleeperStep = Math.max(40, Math.ceil(ROUTE.loopKm / 48));
    for (let km = 0; km <= ROUTE.loopKm; km += sleeperStep) {
      const x = kmToPx(km);
      const sl = document.createElementNS("http://www.w3.org/2000/svg", "line");
      sl.setAttribute("x1", String(x));
      sl.setAttribute("y1", String(RAIL_Y1 - 5));
      sl.setAttribute("x2", String(x));
      sl.setAttribute("y2", String(RAIL_Y2 + 5));
      sl.setAttribute("class", km % 250 === 0 ? "map-sleeper map-sleeper--10" : "map-sleeper");
      railG.appendChild(sl);
    }

    segKmG.textContent = "";
    function addSegKmLabel(kmFrom, kmTo, idx) {
      const mid = (kmFrom + kmTo) / 2;
      const cx = kmToPx(mid);
      const t = document.createElementNS("http://www.w3.org/2000/svg", "text");
      t.setAttribute("x", String(cx));
      t.setAttribute("y", String(30 + (idx % 2) * 18));
      t.setAttribute("text-anchor", "middle");
      t.setAttribute(
        "class",
        "map-seg-km" + (idx % 2 === 1 ? " map-seg-km--zoom" : "")
      );
      t.textContent = kmTo - kmFrom + " км";
      segKmG.appendChild(t);
    }
    const S = ROUTE.settlements;
    for (let i = 0; i < S.length - 1; i++) {
      addSegKmLabel(S[i].km, S[i + 1].km, i);
    }

    restrG.textContent = "";
    ROUTE.segments.forEach(function (seg, ri) {
      if (!seg.restriction) return;
      const w = kmToPx(seg.to) - kmToPx(seg.from);
      if (w < 120) return;
      const cx = (kmToPx(seg.from) + kmToPx(seg.to)) / 2;
      const t = document.createElementNS("http://www.w3.org/2000/svg", "text");
      t.setAttribute("x", String(cx));
      t.setAttribute("y", String(198 + (ri % 2) * 14));
      t.setAttribute("text-anchor", "middle");
      t.setAttribute("class", "map-restr");
      t.textContent = seg.restriction;
      restrG.appendChild(t);
    });

    stations.textContent = "";
    var nSt = ROUTE.settlements.length;
    ROUTE.settlements.forEach(function (st, si) {
      const x = kmToPx(st.km);
      const nc = document.createElementNS("http://www.w3.org/2000/svg", "circle");
      nc.setAttribute("cx", String(x));
      nc.setAttribute("cy", String(RAIL_Y));
      nc.setAttribute("r", st.major ? "7" : "5");
      nc.setAttribute("class", st.major ? "map-node map-node--major" : "map-node");
      nc.setAttribute("title", (st.title || st.name) + " · ~" + Math.round(st.km) + " км");
      stations.appendChild(nc);
      const up = si % 2 === 0;
      const yName = up ? 68 : 152;
      const yKm = up ? 84 : 168;
      const labelAlways =
        si === 0 || si === nSt - 1 || st.major || si % 2 === 0;
      const vis = labelAlways ? "map-np--always" : "map-np--zoom";
      const lab = document.createElementNS("http://www.w3.org/2000/svg", "text");
      lab.setAttribute("x", String(x));
      lab.setAttribute("y", String(yName));
      lab.setAttribute("text-anchor", "middle");
      lab.setAttribute("class", "map-np-name " + vis);
      lab.textContent = st.name;
      stations.appendChild(lab);
      const kmL = document.createElementNS("http://www.w3.org/2000/svg", "text");
      kmL.setAttribute("x", String(x));
      kmL.setAttribute("y", String(yKm));
      kmL.setAttribute("text-anchor", "middle");
      kmL.setAttribute("class", "map-np-km " + vis);
      kmL.textContent = Math.round(st.km) + " км";
      stations.appendChild(kmL);
    });

    if (leg) {
      leg.innerHTML =
        '<span><i style="background:rgba(63,185,80,0.35)"></i> 85+ км/ч</span>' +
        '<span><i style="background:rgba(88,166,255,0.35)"></i> 60–84</span>' +
        '<span><i style="background:rgba(210,153,34,0.35)"></i> 45–59</span>' +
        '<span><i style="background:rgba(248,81,73,0.28)"></i> &lt;45</span>' +
        '<span class="map-legend-note">· узлы — НП; часть подписей и длин перегонов — при увеличении масштаба</span>';
    }
  }

  function setupMapZoom() {
    var svg = document.getElementById("routeSvg");
    var rng = document.getElementById("mapZoom");
    var pctEl = document.getElementById("mapZoomPct");
    var btn = document.getElementById("mapZoomReset");
    var wrap = svg ? svg.closest(".map-svg--scroll") : null;
    if (!svg || !rng) return;
    function apply(pct) {
      var raw = typeof pct === "number" && !isNaN(pct) ? pct : 100;
      var z = Math.max(100, Math.min(330, raw));
      rng.value = String(z);
      if (pctEl) pctEl.textContent = z + "%";
      svg.style.width = z + "%";
      svg.style.height = "";
      if (wrap) wrap.classList.toggle("map-zoom-low", z < 150);
      try {
        localStorage.setItem("mapZoomPct", String(z));
      } catch (_) {}
    }
    var saved = 100;
    try {
      var s = localStorage.getItem("mapZoomPct");
      if (s != null) {
        var p = parseInt(s, 10);
        if (!isNaN(p)) saved = p < 100 ? 100 : p;
      }
    } catch (_) {}
    apply(!isNaN(saved) ? saved : 100);
    rng.addEventListener("input", function () {
      apply(parseInt(rng.value, 10));
    });
    if (btn) {
      btn.addEventListener("click", function () {
        apply(100);
      });
    }
  }

  function initMapUI() {
    initRouteScheme();
    setupMapZoom();
    var rest = document.getElementById("mapRestrictions");
    if (rest) {
      rest.innerHTML = ROUTE.segments
        .map(function (seg) {
          var z = zoneClassV(seg.vMax);
          var note = seg.restriction ? " · " + seg.restriction : "";
          return (
            '<span class="map-seg-chip ' +
            z +
            '" data-from="' +
            seg.from +
            '" data-to="' +
            seg.to +
            '">' +
            seg.from +
            "–" +
            seg.to +
            " км · Vmax " +
            seg.vMax +
            note +
            "</span>"
          );
        })
        .join("");
    }
  }

  function highlightMapSegment(km) {
    document.querySelectorAll(".map-seg-chip").forEach(function (el) {
      var from = parseFloat(el.getAttribute("data-from"), 10);
      var to = parseFloat(el.getAttribute("data-to"), 10);
      var active = km >= from && km < to;
      el.classList.toggle("map-seg-chip--active", active);
    });
  }

  function buildAnalyzePayload(s, mode) {
    const alerts = (s.alerts || []).map(function (a) {
      return (a.code ? "[" + a.code + "] " : "") + (a.text || "");
    });
    const hi = typeof s.health_index === "number" ? s.health_index : 0;
    const coolant = typeof s.coolant_temp_c === "number" ? s.coolant_temp_c : 0;
    const oilT = typeof s.engine_oil_temp_c === "number" ? s.engine_oil_temp_c : 0;
    const factors = (s.health_top_factors || []).map(function (f) {
      return { factor: f.factor || "", penalty: typeof f.penalty === "number" ? f.penalty : 0 };
    });
    return {
      train_id: s.train_id || "",
      timestamp: s.ts || "",
      health_index: hi,
      health_grade: s.health_grade || "",
      health_top_factors: factors,
      speed: typeof s.speed_kmh === "number" ? s.speed_kmh : 0,
      fuel_level: typeof s.fuel_level_l === "number" ? s.fuel_level_l : 0,
      engine_temp: coolant,
      coolant_temp_c: coolant,
      engine_oil_temp_c: oilT,
      traction_motor_temp_c: Array.isArray(s.traction_motor_temp_c) ? s.traction_motor_temp_c.slice() : [],
      brake_pressure: typeof s.brake_pipe_pressure_bar === "number" ? s.brake_pipe_pressure_bar : 0,
      voltage: typeof s.battery_voltage_v === "number" ? s.battery_voltage_v : 0,
      current: typeof s.traction_current_a === "number" ? s.traction_current_a : 0,
      alerts: alerts,
      mode: mode || undefined,
    };
  }

  function sevKey(sev) {
    const t = (sev || "").toLowerCase();
    if (t === "critical" || t === "crit") return "critical";
    if (t === "warning" || t === "warn") return "warning";
    return "normal";
  }

  function renderAIOut(data) {
    if (!aiPanel) return;
    aiPanel.hidden = false;
    const sk = sevKey(data.severity);
    if (aiSeverity) {
      aiSeverity.textContent =
        sk === "critical" ? "критично" : sk === "warning" ? "внимание" : "норма";
      aiSeverity.setAttribute("data-sev", sk);
    }
    if (aiSummary) aiSummary.textContent = data.summary || "—";

    function fillList(ul, items) {
      if (!ul) return;
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
    if (aiMetrics) {
      aiMetrics.textContent = am.length > 0 ? "Метрики: " + am.join(", ") : "Метрики: —";
    }

    if (aiRisk) {
      if (data.next_risk && String(data.next_risk).trim()) {
        aiRisk.hidden = false;
        aiRisk.textContent = "Риск далее: " + data.next_risk;
      } else {
        aiRisk.hidden = true;
        aiRisk.textContent = "";
      }
    }
  }

  function setAILoading(loading) {
    if (btnAIExplain) btnAIExplain.disabled = loading;
    if (btnAIActions) btnAIActions.disabled = loading || !aiEnabled;
    if (aiStatusLine) {
      aiStatusLine.textContent = loading
        ? "запрос к модели…"
        : aiEnabled
          ? "ИИ готов (on-demand, без каждого тика телеметрии)."
          : "ИИ отключён: задайте GEMINI_API_KEY или OPENAI_API_KEY на сервере.";
    }
  }

  async function runAI(mode) {
    if (!aiEnabled) {
      if (aiErr) {
        aiErr.textContent = "Сервер без GEMINI_API_KEY / OPENAI_API_KEY — см. README.";
        aiErr.hidden = false;
      }
      return;
    }
    if (!lastSample || !lastSample.ts) {
      if (aiErr) {
        aiErr.textContent = "Нет данных телеметрии для выбранного поезда.";
        aiErr.hidden = false;
      }
      return;
    }
    if (aiErr) aiErr.hidden = true;
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
      if (aiErr) {
        aiErr.textContent = "Ошибка: " + (e && e.message ? e.message : String(e));
        aiErr.hidden = false;
      }
    } finally {
      setAILoading(false);
    }
  }

  async function refreshAIStatus() {
    if (!aiStatusLine) return;
    try {
      const res = await fetch("/api/v1/ai/status", fetchOpts());
      if (!res.ok) throw new Error("status");
      const j = await res.json();
      aiEnabled = !!j.enabled;
      aiStatusLine.textContent = aiEnabled
        ? "ИИ готов (on-demand, без каждого тика телеметрии)."
        : "ИИ отключён: задайте GEMINI_API_KEY или OPENAI_API_KEY на сервере.";
      if (btnAIExplain) btnAIExplain.disabled = false;
      if (btnAIActions) btnAIActions.disabled = !aiEnabled;
    } catch (_) {
      aiEnabled = false;
      aiStatusLine.textContent = "Не удалось проверить /api/v1/ai/status";
      if (btnAIActions) btnAIActions.disabled = true;
    }
  }

  function updateMap(s) {
    const mileage = typeof s.mileage_km === "number" ? s.mileage_km : 0;
    const km = ((mileage % ROUTE.loopKm) + ROUTE.loopKm) % ROUTE.loopKm;
    const cx = kmToPx(km);
    if (mapDot) {
      mapDot.setAttribute("cx", cx.toFixed(1));
      mapDot.setAttribute("cy", String(RAIL_Y));
    }

    const seg = findSegment(km);
    const vMax = seg.vMax;
    const spd = typeof s.speed_kmh === "number" ? s.speed_kmh : 0;
    const over = spd > vMax + 2;

    if (mapDot) mapDot.classList.toggle("map-dot--warn", over);
    if (mapCaption) mapCaption.classList.toggle("map-caption--warn", over);

    const g = geoAtKm(km);
    const geo = g.lat.toFixed(5) + "°, " + g.lng.toFixed(5) + "° · ";
    if (mapCaption) {
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
    }

    if (mapSubcaption) {
      const leg = routeLegAt(km);
      let sub = leg.wrap
        ? "между " + leg.from.name + " и " + leg.to.name + " (замыкание кольца) · " + leg.legKm + " км"
        : "между " + leg.from.name + " и " + leg.to.name + " · " + leg.legKm + " км";
      sub += " · Vmax " + vMax + " км/ч";
      if (seg.restriction) sub += " · " + seg.restriction;
      mapSubcaption.textContent = sub;
    }

    highlightMapSegment(km);
  }

  function clearBuffers() {
    buffers.labels = [];
    buffers.speed = [];
    buffers.coolant = [];
    buffers.health = [];
    buffers.fuel = [];
  }

  function pushBuffers(s) {
    const label = new Date(s.ts || Date.now()).toLocaleTimeString();
    buffers.labels.push(label);
    buffers.speed.push(s.speed_kmh ?? null);
    buffers.coolant.push(s.coolant_temp_c ?? null);
    buffers.health.push(s.health_index ?? null);
    buffers.fuel.push(s.fuel_level_l ?? null);
    while (buffers.labels.length > MAX_POINTS) {
      buffers.labels.shift();
      buffers.speed.shift();
      buffers.coolant.shift();
      buffers.health.shift();
      buffers.fuel.shift();
    }
  }

  function cssVar(name, fallback) {
    const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function chartOpts(title) {
    const grid = cssVar("--border", "#30363d");
    const text = cssVar("--muted", "#8b949e");
    const plugins = {
      legend: { display: false },
      title: { display: true, text: title, color: text, font: { size: 11 } },
    };
    if (typeof ChartZoom !== "undefined") {
      plugins.zoom = {
        pan: { enabled: true, mode: "x", modifierKey: null },
        zoom: {
          wheel: { enabled: true },
          pinch: { enabled: true },
          mode: "x",
        },
      };
    }
    return {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: "index", intersect: false },
      plugins: plugins,
      scales: {
        x: {
          display: buffers.labels.length > 2,
          ticks: { maxTicksLimit: 12, color: text },
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
    if (typeof ChartZoom !== "undefined") {
      try {
        Chart.register(ChartZoom);
      } catch (e) {
        console.warn("chartjs-plugin-zoom register", e);
      }
    }
    const accent = cssVar("--accent", "#58a6ff");
    const ok = cssVar("--ok", "#3fb950");
    const warn = cssVar("--warn", "#f0883e");
    const fuelCol = cssVar("--chart-fuel", "#c9a227");

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
    charts.health.options.scales.y.min = 0;
    charts.health.options.scales.y.max = 100;
    charts.fuel = new Chart(document.getElementById("chartFuel"), {
      type: "line",
      data: {
        labels: [],
        datasets: [{ label: "л", data: [], borderColor: fuelCol, tension: 0.25, fill: false }],
      },
      options: chartOpts("Топливо"),
    });
  }

  function syncChartTheme() {
    if (!charts.speed) return;
    const accent = cssVar("--accent", "#58a6ff");
    const ok = cssVar("--ok", "#3fb950");
    const warn = cssVar("--warn", "#f0883e");
    const fuelCol = cssVar("--chart-fuel", "#c9a227");
    const grid = cssVar("--border", "#30363d");
    const text = cssVar("--muted", "#8b949e");
    charts.speed.data.datasets[0].borderColor = accent;
    charts.temp.data.datasets[0].borderColor = warn;
    charts.health.data.datasets[0].borderColor = ok;
    charts.fuel.data.datasets[0].borderColor = fuelCol;
    ["speed", "temp", "health", "fuel"].forEach(function (k) {
      const ch = charts[k];
      ch.options.plugins.title.color = text;
      ch.options.scales.x.ticks.color = text;
      ch.options.scales.y.ticks.color = text;
      ch.options.scales.x.grid.color = grid;
      ch.options.scales.y.grid.color = grid;
      ch.update();
    });
  }

  function updateCharts() {
    if (!charts.speed) return;
    const series = [
      ["speed", buffers.speed],
      ["temp", buffers.coolant],
      ["health", buffers.health],
      ["fuel", buffers.fuel],
    ];
    series.forEach(function (row) {
      const k = row[0];
      const ds = row[1];
      const ch = charts[k];
      ch.data.labels = buffers.labels.slice();
      ch.data.datasets[0].data = ds.slice();
      ch.options.scales.x.display = buffers.labels.length > 2;
      const ys = ch.options.scales && ch.options.scales.y;
      if (ys) {
        if (k === "health") {
          ys.min = 0;
          ys.max = 100;
        } else {
          ys.min = undefined;
          ys.max = undefined;
        }
      }
      ch.update();
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
      if (!replayMode) setConnected(true, "онлайн");
    };
    ws.onclose = function () {
      if (!replayMode) {
        setConnected(false, "нет связи · переподключение…");
        reconnectTimer = setTimeout(connectWS, backoffMs);
        backoffMs = Math.min(backoffMs * 2, 30000);
      }
    };
    ws.onerror = function () {
      try {
        ws.close();
      } catch (_) {}
    };
    ws.onmessage = function (ev) {
      if (replayMode) return;
      const p =
        typeof ev.data === "string"
          ? Promise.resolve(ev.data)
          : ev.data instanceof Blob
            ? ev.data.text()
            : Promise.resolve(String(ev.data));
      p.then(function (raw) {
        const s = JSON.parse(raw);
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

  function stopLivePoll() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  function pollLatest() {
    if (replayMode) return;
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
    stopLivePoll();
    pollTimer = setInterval(pollLatest, 1000);
    pollLatest();
  }

  function stopReplayPlay() {
    playing = false;
    btnReplayPlay.textContent = "▶";
    if (playTimer) {
      clearInterval(playTimer);
      playTimer = null;
    }
  }

  async function loadHistoryLive() {
    const tid = encodeURIComponent(trainId());
    const min = windowMinutes();
    const res = await fetch(
      "/api/v1/telemetry/history?train_id=" + tid + "&minutes=" + min,
      fetchOpts()
    );
    if (!res.ok) return;
    const list = await res.json();
    clearBuffers();
    if (!Array.isArray(list) || list.length === 0) {
      const r2 = await fetch("/api/v1/telemetry/latest?train_id=" + tid, fetchOpts());
      if (r2.ok) {
        const s = await r2.json();
        applySample(s);
      }
      return;
    }
    list.forEach(function (s) {
      pushBuffers(s);
    });
    const last = list[list.length - 1];
    lastAppliedTs = last.ts || "";
    renderDashboard(last);
    try {
      updateCharts();
    } catch (e) {
      console.warn("chart update", e);
    }
  }

  async function loadReplayWindow() {
    const tid = encodeURIComponent(trainId());
    const min = windowMinutes();
    const res = await fetch(
      "/api/v1/telemetry/history?train_id=" + tid + "&minutes=" + min,
      fetchOpts()
    );
    if (!res.ok) {
      replayTimeLabel.textContent = "ошибка загрузки";
      return;
    }
    const list = await res.json();
    if (!Array.isArray(list) || list.length === 0) {
      replaySamples = [];
      replayScrubWrap.hidden = true;
      replayTimeLabel.textContent = "нет данных за окно";
      return;
    }
    replayMode = true;
    stopReplayPlay();
    stopLivePoll();
    setConnected(false, "replay");
    try {
      if (ws) ws.close();
    } catch (_) {}
    ws = null;

    replaySamples = list;
    clearBuffers();
    list.forEach(function (s) {
      pushBuffers(s);
    });
    replayIndex = list.length - 1;
    replaySlider.max = String(Math.max(0, list.length - 1));
    replaySlider.value = String(replayIndex);
    replayScrubWrap.hidden = false;
    try {
      updateCharts();
    } catch (e) {
      console.warn("chart update", e);
    }
    applyReplayFrame(replayIndex);
  }

  function exitReplay() {
    replayMode = false;
    stopReplayPlay();
    replayScrubWrap.hidden = true;
    replaySamples = [];
    const liveRadio = document.querySelector('input[name="viewMode"][value="live"]');
    if (liveRadio) liveRadio.checked = true;
    startLivePoll();
    connectWS();
    loadHistoryLive().catch(function () {});
  }

  function setViewMode(mode) {
    if (mode === "live") {
      exitReplay();
    } else {
      btnLoadReplay.focus();
    }
  }

  document.querySelectorAll('input[name="viewMode"]').forEach(function (el) {
    el.addEventListener("change", function () {
      if (el.value === "live" && el.checked) setViewMode("live");
      if (el.value === "replay" && el.checked) setViewMode("replay");
    });
  });

  replaySlider.addEventListener("input", function () {
    if (!replayMode) return;
    stopReplayPlay();
    applyReplayFrame(parseInt(replaySlider.value, 10) || 0);
  });

  btnLoadReplay.addEventListener("click", function () {
    document.querySelector('input[name="viewMode"][value="replay"]').checked = true;
    loadReplayWindow().catch(function (e) {
      console.warn(e);
    });
  });

  btnReplayPlay.addEventListener("click", function () {
    if (!replayMode || replaySamples.length < 2) return;
    if (playing) {
      stopReplayPlay();
      return;
    }
    playing = true;
    btnReplayPlay.textContent = "■";
    playTimer = setInterval(function () {
      if (replayIndex >= replaySamples.length - 1) {
        stopReplayPlay();
        return;
      }
      applyReplayFrame(replayIndex + 1);
    }, 250);
  });

  trainInput.addEventListener("change", function () {
    clearBuffers();
    document.querySelector('input[name="viewMode"][value="live"]').checked = true;
    exitReplay();
  });

  replayMinutesEl.addEventListener("change", function () {
    if (!replayMode) {
      loadHistoryLive().catch(function () {});
    }
  });

  btnTheme.addEventListener("click", function () {
    const root = document.documentElement;
    const t = root.getAttribute("data-theme") === "light" ? "dark" : "light";
    root.setAttribute("data-theme", t);
    requestAnimationFrame(syncChartTheme);
  });

  if (btnAIExplain) btnAIExplain.addEventListener("click", function () { runAI(""); });
  if (btnAIActions) btnAIActions.addEventListener("click", function () { runAI("actions"); });

  document.addEventListener("DOMContentLoaded", function () {
    initMapUI();
    refreshAIStatus().catch(function () {});
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
    document.addEventListener("visibilitychange", function () {
      if (document.visibilityState === "visible" && !replayMode) {
        pollLatest();
      }
    });

    loadHistoryLive()
      .catch(function () {})
      .then(function () {
        connectWS();
        startLivePoll();
      });
  });
})();