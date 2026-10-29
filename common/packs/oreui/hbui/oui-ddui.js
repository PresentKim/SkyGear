// Draws a server-driven DDUI screen in place of the vanilla custom form.
//
// The server opens minecraft:custom_form and puts its own fields next to the vanilla ones in data
// store "minecraft":
//   custom_form_data_1.oui        JSON {form, root}: the layout tree and the id of the form, which its
//                                 variable patches carry too. A new layout of the form shown is
//                                 compared with the one drawn and only the nodes that differ change
//                                 (patch); another form's is drawn anew
//   custom_form_data_1.oui_event  written here: JSON {seq, id, kind, value} for every interaction
//   oui_vars_1_<n>.v              JSON {f, s, p}: patch p of changed variables, merged into what the
//                                 renderer has. Patches rotate over VARS_SLOTS properties, since the
//                                 client keeps only the newest value of a property within a frame;
//                                 each is applied once, and only for the form f shown.
// Reads go through the same engine queries the vanilla bundle uses (vanilla.ddui.string,
// vanilla.ddui.screenId); the write goes through the vanilla DDUI command group.
//
// Layout nodes are {t, id, text, bind, class, style, tip, children, ...}:
//   col, row, grid (cols), text, button (active, activeBind, disabled), bar (bind 0..1), toggle, slider
//   (min, max, step), input (placeholder), select (options), item (icon, size, count, text), slot (bind
//   to {id, n}, size), region (name), watch (property, path: shows a string the data store holds, read
//   through its own query), space.
// More nodes for the Minecraft style (oui-mc.css): frame (an empty slot), arrow, img (icon, size), plane
// (fov, dist: boxes with `at` [x, z, w, h] in blocks over the ground a straight-down camera shows).
// For trying what an inventory needs: slot with `raw` (rawSlot), box with `self` (presses on the box
// itself, any button), cursor (bind: the item on the pointer), probe (input events and engine command
// groups) and engine (slots [[container, index]], sizes, calls [{label, cmd, args}]: the vanilla
// container queries and commands).
// `ink: "panel"` on a node makes the text under it the dark variant of its § colours (text drawn right on
// the grey window); buttons, slot counts and tooltips keep the colours.
// The body gets mc-w-s / mc-w-m / mc-w-l for the window width, so the CSS can lay out by size.
// `bind` names a variable that replaces the text, or the value of a bar. Text may carry § colour codes.
// An icon is "geo:<item name>" (drawn by the engine through geo-item://, the URL the vanilla container
// slots use), "tex:<texture path>" (a pack texture through pack://), or a bare item name.
// A region's children come from the variable "@<name>" (a node), so the server can replace one part of
// the screen without resending the layout; the region is patched like the layout. `tip` shows a tooltip while the pointer is over the node.
// `alt` on a box is the id its press with the right mouse button sends.
// `end` on a scrolling box keeps it scrolled to its end after each draw (a log).
// pixels (w, h, px: w*h colours "rrggbb", "" for none) is a small picture drawn as boxes (a skin's face).
// `keys` on the root ({up, down, left, right, ok: id}) presses the id for the arrow keys or WASD, Enter or
// the space bar (onKey).
(function () {
  var STORE = "minecraft";
  var PROPERTY = "custom_form_data_1";
  var VARS_PROPERTY = "oui_vars_1_";
  var VARS_SLOTS = 8;
  var SCREEN = "minecraft:custom_form";
  var engine = window.engine;
  var applied = {}; // seq numbers of the patches applied for the form shown
  var held = []; // patches of a form whose layout has not arrived yet

  var state = { screen: "", form: "", layout: null, layoutText: "", vars: {}, seq: 0, updates: 0, error: "" };
  var bound = []; // {el, key, kind, region}
  var regions = {}; // name -> element
  var regionInk = {}; // name -> the ink where the region sits
  var regionRecs = {}; // name -> the record of what the region shows
  var tree = null; // the record of the layout shown
  var building = ""; // the region being built, so its bindings can be dropped when it is rebuilt
  var bars = []; // {sc, bar, handle}: the scrollbars of the mc-scroll elements shown
  var root = null;
  var status = null;
  var tip = null;

  // --- engine access -------------------------------------------------------------------------

  var nextQuery = 0x7e000000;
  function subscribe(name, params, cb) {
    var id = nextQuery++;
    engine.on("query:subscribed/" + id, cb);
    engine.on("query:updated/" + id, cb);
    engine.trigger.apply(engine, ["query:subscribe/" + name, id].concat(params));
  }

  function unwrap(v) {
    if (v !== null && typeof v === "object" && "value" in v) return v.value;
    return v;
  }

  function command(group, name) {
    var g = window.__commands__ && window.__commands__[group];
    var c = g && g[name];
    if (!c || !c.callable) return null;
    return function () {
      return c.callable.apply(c, arguments);
    };
  }

  function send(id, kind, value) {
    var set = command("vanillaDduiDataStoreSetCommandGroup", "setDataStorePathString");
    state.seq++;
    var msg = { seq: state.seq, id: id, kind: kind, value: value === undefined ? null : value, at: kind === "click" ? lastPress : null };
    // A press carries the text of the input boxes, so a button that submits them reads what was typed
    // even before the box's own change event.
    if (kind === "click") msg.inputs = inputValues();
    var ev = JSON.stringify(msg);
    if (!set) {
      state.error = "setDataStorePathString not found";
      drawStatus();
      return;
    }
    try {
      set(STORE, PROPERTY, "oui_event", ev);
    } catch (e) {
      state.error = "send: " + e;
    }
    drawStatus();
  }

  // --- client translation ----------------------------------------------------------------------
  // The server sends names and dates as client spans (common/i18n ClientKey, ClientDate, ClientAgo):
  // U+FDD0 kind:value U+FDD1 fallback U+FDD2. localize puts the client's own text in their place: tr for
  // a translation key the client knows (the vanilla and the pack's texts/*.lang), fmtDate and fmtAgo for
  // a moment in Unix seconds, in the device's language. Each shows the server's text (in the player's
  // language) when the engine has no answer. Other pages reach them as window.__oui.

  var TRANSLATE = "coreTranslateCommandGroup";
  var translated = {}; // key -> the client's text, null: none

  // engineText is the text the engine command name of the translate group gives for arg, or null.
  function engineText(name, arg) {
    var f = command(TRANSLATE, name);
    if (!f) return undefined;
    var v;
    try {
      v = unwrap(f(arg));
    } catch (e) {
      return null;
    }
    return typeof v === "string" && v !== "" ? v : null;
  }

  function tr(key, fallback) {
    if (fallback === undefined) fallback = key;
    if (!(key in translated)) {
      var v = engineText("translate", key);
      if (v === undefined) return fallback; // not ready: ask again next time
      translated[key] = v === key ? null : v;
    }
    return translated[key] === null ? fallback : translated[key];
  }

  function fmtDate(sec, fallback) {
    var v = engineText("formatDate", Number(sec));
    return v ? v : fallback;
  }

  function fmtAgo(sec, fallback) {
    var v = engineText("getHowLongAgoAsString", Number(sec));
    return v ? v : fallback;
  }

  var SPAN_OPEN = "\uFDD0";
  var SPAN = /\uFDD0([a-z]):([^\uFDD1]*)\uFDD1([^\uFDD2]*)\uFDD2/g;

  function localize(s) {
    if (typeof s !== "string" || s.indexOf(SPAN_OPEN) < 0) return s;
    return s.replace(SPAN, function (_, kind, value, fallback) {
      if (kind === "k") return tr(value, fallback);
      if (kind === "d") return fmtDate(value, fallback);
      if (kind === "a") return fmtAgo(value, fallback);
      return fallback;
    });
  }

  window.__oui = window.__oui || {};
  window.__oui.tr = tr;
  window.__oui.fmtDate = fmtDate;
  window.__oui.fmtAgo = fmtAgo;
  window.__oui.localize = localize;

  // --- text ------------------------------------------------------------------------------------

  var COLOURS = {
    "0": "#000000", "1": "#0000aa", "2": "#00aa00", "3": "#00aaaa", "4": "#aa0000", "5": "#aa00aa", "6": "#ffaa00",
    "7": "#aaaaaa", "8": "#555555", "9": "#5555ff", a: "#55ff55", b: "#55ffff", c: "#ff5555", d: "#ff55ff",
    e: "#ffff55", f: "#ffffff", g: "#ddd605"
  };

  // On the grey window the § colours, written for a dark background, become their dark variants.
  var ON_PANEL = { f: "0", "7": "8", "8": "7", e: "1", "6": "4", a: "2", b: "3", c: "4", d: "5" };
  var ink = "dark"; // the ink of the node being built

  // rich fills e with s, turning § colour and format codes into spans and newlines into lines.
  function rich(e, s, panel) {
    if (panel === undefined) panel = ink === "panel";
    e.innerHTML = "";
    s = s === undefined || s === null ? "" : localize(String(s));
    s.split("\n").forEach(function (line, i) {
      var lineEl = el("div", "oui-line");
      // Gameface lays the colour runs of a line out as flex items aligned at their tops (no baseline
      // alignment), so a run the Minecraft font draws sits higher than one in the locale's Noto font.
      // A line with any non-ASCII letter is drawn in the Noto font throughout.
      if (/[^\x00-\x7f§]/.test(line)) lineEl.className += " oui-wide";
      var colour = "", bold = false, italic = false, run = "";
      function flush() {
        if (!run) return;
        var span = el("span", null, run);
        if (colour) span.style.color = colour;
        if (bold) span.style.fontWeight = "bold";
        if (italic) span.style.fontStyle = "italic";
        lineEl.appendChild(span);
        run = "";
      }
      for (var j = 0; j < line.length; j++) {
        var c = line.charAt(j);
        if (c === "§" && j + 1 < line.length) {
          flush();
          var code = line.charAt(++j).toLowerCase();
          if (panel && ON_PANEL[code]) code = ON_PANEL[code];
          if (code === "k") continue;
          if (COLOURS[code]) colour = COLOURS[code];
          else if (code === "l") bold = true;
          else if (code === "o") italic = true;
          else if (code === "r") {
            colour = "";
            bold = italic = false;
          }
          continue;
        }
        run += c;
      }
      flush();
      if (!lineEl.firstChild) lineEl.textContent = " ";
      e.appendChild(lineEl);
    });
  }

  // --- rendering -------------------------------------------------------------------------------

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined && text !== null) e.textContent = String(text);
    return e;
  }

  // Gameface does not deliver "click" to plain divs, so presses fire on the left mouse button going down.
  // A touch has no pointer resting over an element to show its tooltip, so on a touch screen the first
  // tap on an element with a tooltip (nodeOf: its node, whose tip is read at the tap) shows the tooltip
  // and a second tap on it presses it, like the JSON UI forms. A node with `quick` (a back button, a
  // tab) presses at once. The tap is told by the touch events around the mousedown, whose order the
  // devices do not agree on, so the decision waits a moment. The mousedown bubbles up through the
  // elements that listen, so only the innermost one that takes presses (wants, when given) answers it.
  function onPress(e, fn, nodeOf, wants) {
    pressables.push({ e: e, fn: fn, nodeOf: nodeOf });
    e.addEventListener("mousedown", function (ev) {
      if (ev.button !== undefined && ev.button !== 0) return;
      if (claimed === ev || (wants && !wants())) return;
      claimed = ev;
      lastPress = { x: Math.round(ev.clientX), y: Math.round(ev.clientY), w: window.innerWidth, h: window.innerHeight };
      var at = Date.now();
      setTimeout(
        function () {
          var n = nodeOf ? nodeOf() : null;
          var touch = touchSeen || Math.abs(touchAt - at) < TOUCH_WINDOW;
          if (touch && n && n.tip && !n.quick && armed !== e) {
            armed = e;
            showTip(e, n.tip);
            return;
          }
          armed = null;
          if (touch) hideTip();
          fn(ev);
        },
        touchSeen ? 0 : 40
      );
    });
  }
  var lastPress = null;
  var pressables = []; // {e, fn, nodeOf}: the elements onPress made pressable, for the keyboard selection
  var claimed = null; // the mousedown an element answered
  var TOUCH_WINDOW = 700; // ms between a touch event and the mousedown of the same tap
  var touchAt = 0; // when the last touch began
  var touchSeen = false; // whether the presses come from a touch screen (until the mouse moves alone)
  var armed = null; // the element whose tooltip the last tap showed; the next tap on it presses it

  // noteTouch keeps when a touch began and puts away the tooltip of an element the touch is not on.
  function noteTouch(ev) {
    touchAt = Date.now();
    touchSeen = true;
    if (armed && !armed.contains(ev.target)) {
      armed = null;
      hideTip();
    }
  }

  // A mouse that moves with no touch about is a mouse: presses go through at once again.
  function noteMouse() {
    if (touchSeen && Date.now() - touchAt > TOUCH_WINDOW) {
      touchSeen = false;
      armed = null;
    }
  }

  // pointOf is where a mouse or touch event happened. A touch device sends a mousedown for a tap but
  // no mousemove while the finger moves, so drags also listen to the touch events.
  function pointOf(ev) {
    var t = (ev.touches && ev.touches[0]) || (ev.changedTouches && ev.changedTouches[0]);
    return t || ev;
  }

  // onDragStart runs fn on the mouse button or a finger going down on e.
  function onDragStart(e, fn) {
    e.addEventListener("mousedown", fn);
    e.addEventListener("touchstart", fn);
  }

  function bind(entry) {
    entry.region = building;
    if (entry.panel === undefined) entry.panel = ink === "panel";
    bound.push(entry);
  }

  function applyCommon(e, n, rec) {
    if (n["class"]) e.className += " " + n["class"];
    if (n.style) for (var k in n.style) e.style[k] = n.style[k];
    rec.inner = e;
    if (n.tip) withTip(e, rec);
  }

  function varOr(key, fallback) {
    return key && Object.prototype.hasOwnProperty.call(state.vars, key) ? state.vars[key] : fallback;
  }

  function iconSrc(icon, size) {
    icon = String(icon || "");
    if (icon.indexOf("tex:") === 0) return "pack://" + icon.slice(4) + ".png?filtering=point";
    // geo-item:// looks the host up by item name, then draws "#item_id_aux" from the bag instead when
    // the bag has it (OreUI::ItemResourceResolver, GeometryAtlas::ItemHandleFactoryImpl): any valid
    // name works as the host, and the id draws custom items and blocks like JSON UI's item renderer.
    if (icon.indexOf("aux:") === 0) {
      var bag = encodeURIComponent('{"#item_id_aux":' + icon.slice(4) + "}");
      return "geo-item://stone?width=" + size + "&height=" + size + "&bag=" + bag;
    }
    if (icon.indexOf("geo:") === 0) icon = icon.slice(4);
    return "geo-item://" + icon + "?width=" + size + "&height=" + size;
  }

  // withTip shows the tooltip of the node of rec (read when the pointer comes, so a node updated in
  // place shows its new tooltip) while the pointer is over e.
  function withTip(e, rec) {
    rec.tipped = true;
    e.addEventListener("mouseover", function () {
      if (panning || !rec.n.tip) return;
      showTip(e, rec.n.tip);
    });
    e.addEventListener("mouseout", function () {
      // The tooltip a tap showed stays until the next tap.
      if (armed !== e) hideTip();
    });
  }
  var tipOwner = null; // the element whose tooltip shows

  // showTip shows text as the tooltip of e, right of it, or left of it when there is no room at the
  // right, kept inside the screen. After a touch it is a step larger (#oui-tip.touch).
  function showTip(e, text) {
    if (!tip) {
      tip = el("div");
      tip.id = "oui-tip";
      document.body.appendChild(tip);
    }
    tipOwner = e;
    tip.className = touchSeen ? "touch" : "";
    rich(tip, text, false);
    tip.style.left = "0px";
    tip.style.top = "0px";
    tip.style.display = "flex";
    var r = e.getBoundingClientRect();
    var t = tip.getBoundingClientRect();
    var vw = window.innerWidth, vh = window.innerHeight, gap = 6;
    var x = r.right + gap;
    if (x + t.width > vw - gap) x = r.left - gap - t.width;
    x = Math.max(gap, Math.min(x, vw - gap - t.width));
    var y = Math.max(gap, Math.min(r.top, vh - gap - t.height));
    tip.style.left = Math.round(x) + "px";
    tip.style.top = Math.round(y) + "px";
  }

  function hideTip() {
    if (tip) tip.style.display = "none";
    tipOwner = null;
  }

  // A built node keeps a record: the node (n), the element that sits in its parent (el: the scroll
  // wrapper of an mc-scroll element), the element the class, style and tooltip went on (inner), the
  // element its children go in (box, with each child in a cell for a grid), the records of its
  // children (kids) and the ink it was built with. A layout that comes again for the same form is
  // compared with the records (patch), so only what changed is drawn again: the window, the scroll
  // positions, the focused input and the images loaded stay.
  function make(n) {
    var outerInk = ink;
    if (n.ink) ink = n.ink;
    var rec = { n: n, el: null, inner: null, box: null, grid: false, kids: [], ink: ink, tipped: false };
    try {
      rec.el = buildNode(n, rec);
    } finally {
      ink = outerInk;
    }
    return rec;
  }

  function buildNode(n, rec) {
    var e;
    switch (n.t) {
      case "col":
      case "row":
        e = el("div", "oui-" + n.t);
        if (n.scroll) e.className += " oui-scroll";
        appendChildren(e, n.children, rec);
        break;
      case "grid":
        e = el("div", "oui-grid");
        rec.box = e;
        rec.grid = true;
        (n.children || []).forEach(function (c) {
          var r = make(c);
          rec.kids.push(r);
          e.appendChild(gridCell(n, r.el));
        });
        break;
      case "text":
        e = el("div", "oui-text");
        rich(e, varOr(n.bind, n.text));
        if (n.bind) bind({ el: e, key: n.bind, kind: "text" });
        break;
      case "button":
        e = el("div", "oui-button");
        if (n.icon) {
          var bimg = el("img");
          bimg.src = iconSrc(n.icon, 32);
          e.appendChild(bimg);
        }
        var label = el("span", "oui-label-text");
        rich(label, varOr(n.bind, n.text), false);
        e.appendChild(label);
        if (n.active) e.className += " on";
        if (n.disabled) e.className += " disabled";
        if (n.bind) bind({ el: label, key: n.bind, kind: "text", panel: false });
        if (n.activeBind) {
          if (varOr(n.activeBind, false)) e.className += " on";
          bind({ el: e, key: n.activeBind, kind: "on" });
        }
        if (!n.disabled) {
          onPress(
            e,
            function () {
              send(rec.n.id, "click", null);
            },
            function () {
              return rec.n;
            }
          );
        }
        break;
      case "bar":
        e = el("div", "oui-bar");
        var fill = el("div", "oui-bar-fill");
        fill.style.width = Math.round(100 * Number(varOr(n.bind, n.value || 0))) + "%";
        e.appendChild(fill);
        if (n.bind) bind({ el: fill, key: n.bind, kind: "width" });
        break;
      case "toggle":
        e = buildToggle(n);
        break;
      case "slider":
        e = buildSlider(n);
        break;
      case "input":
        e = buildInput(n);
        break;
      case "select":
        e = buildSelect(n);
        break;
      case "item":
        e = buildItem(n);
        break;
      case "slot":
        e = buildSlot(n);
        break;
      case "region":
        e = el("div", "oui-region");
        regions[n.name] = e;
        regionInk[n.name] = ink;
        delete regionRecs[n.name];
        fillRegion(n.name);
        break;
      case "watch":
        e = el("div", "oui-text", "…");
        (function (target) {
          subscribe("vanilla.ddui.string", [STORE, n.property, n.path, false], function (v) {
            v = unwrap(v);
            rich(target, typeof v === "string" ? v : JSON.stringify(v));
          });
        })(e);
        break;
      case "box":
        // A plain div for composed components: classes, children, a press (when it has an id and
        // press is not false) and activeBind, which toggles the "on" class.
        e = el("div", "oui-box");
        appendChildren(e, n.children, rec);
        if (n.activeBind) {
          if (varOr(n.activeBind, false)) e.className += " on";
          bind({ el: e, key: n.activeBind, kind: "on" });
        }
        if (n.on) e.className += " on";
        // `end` keeps a scrolling box at its end (a log): scrollToEnds after each draw.
        if (n.end) e.className += " oui-end";
        // `alt` is the id the right mouse button presses.
        e.addEventListener("mousedown", function (ev) {
          if (ev.button !== 2 || !rec.n.alt || claimed === ev) return;
          claimed = ev;
          send(rec.n.alt, "click", null);
        });
        // A `self` box takes the presses of any button that land on the box itself and not on what it
        // holds: the backdrop around a window, where the vanilla inventory drops the held item.
        if (n.self) {
          e.addEventListener("mousedown", function (ev) {
            if (ev.target === e && rec.n.id) send(rec.n.id, "click", { b: ev.button || 0 });
          });
          break;
        }
        // The press reads the node of the record, so a box updated in place may gain or lose it. A box
        // that does nothing when pressed still shows its tooltip at a tap.
        onPress(
          e,
          function () {
            if (rec.n.id && rec.n.press !== false) send(rec.n.id, "click", null);
          },
          function () {
            return rec.n;
          },
          function () {
            return !!((rec.n.id && rec.n.press !== false) || rec.n.tip);
          }
        );
        break;
      case "frame":
        e = el("div", "mc-frame");
        break;
      case "arrow":
        e = el("div", "mc-arrow");
        break;
      case "img":
        e = el("img", "oui-img");
        e.src = iconSrc(n.icon, n.size || 64);
        break;
      case "space":
        e = el("div", "oui-space");
        break;
      case "pixels":
        // A row of boxes for each row of the picture: the page has no canvas (getContext). Rows and boxes
        // take their share of the picture's size in percent: Gameface does not stretch empty flex items.
        e = el("div", "oui-pixels");
        for (var py = 0; py < (n.h || 0); py++) {
          var prow = el("div", "oui-pixels-row");
          prow.style.height = 100 / n.h + "%";
          for (var px = 0; px < (n.w || 0); px++) {
            var cell = el("div", "oui-pixel");
            cell.style.width = 100 / n.w + "%";
            var colour = (n.px || [])[py * n.w + px];
            if (colour) cell.style.backgroundColor = "#" + colour;
            prow.appendChild(cell);
          }
          e.appendChild(prow);
        }
        break;
      case "canvas":
        e = buildCanvas(n, rec);
        break;
      case "plane":
        // The ground a straight-down camera shows: the children are boxes placed by `at` [x, z, w, h]
        // in blocks from the point under the camera (x east, z south), scaled from the camera's vertical
        // field of view `fov` (degrees) and its height `dist` over the ground (layoutPlanes).
        e = el("div", "oui-plane");
        e.__rec = rec;
        appendChildren(e, n.children, rec);
        setTimeout(layoutPlanes, 0);
        break;
      case "line":
        // A straight line from (x1, y1) to (x2, y2), in rem: a thin box turned around its start.
        e = el("div", "oui-line-seg");
        var dx = n.x2 - n.x1, dy = n.y2 - n.y1;
        e.style.left = n.x1 + "rem";
        e.style.top = n.y1 + "rem";
        e.style.width = Math.sqrt(dx * dx + dy * dy) + "rem";
        e.style.transform = "rotate(" + (Math.atan2(dy, dx) * 180) / Math.PI + "deg)";
        break;
      case "cursor":
        e = buildCursor(n);
        break;
      case "probe":
        e = el("div", "oui-probe");
        probeEl = e;
        envSent = false;
        drawProbe();
        break;
      case "engine":
        e = buildEngine(n);
        break;
      case "restest":
        e = buildResTest(n);
        break;
      default:
        e = el("div", "oui-text", "unknown node " + n.t);
    }
    applyCommon(e, n, rec);
    if (isScroll(n)) return withScrollbar(e);
    return e;
  }

  // withScrollbar puts the vanilla mouse scrollbar (ui_common scroll_box: ScrollRail track, ScrollHandle
  // box) beside a scrolling element. Gameface draws no scrollbar of its own; the bar shows only while the
  // content is taller than the element (updateBar).
  function withScrollbar(sc) {
    var wrap = el("div", "mc-scroll-wrap");
    var bar = el("div", "mc-scrollbar");
    var handle = el("div", "mc-scroll-handle");
    bar.appendChild(el("div", "mc-scroll-rail"));
    bar.appendChild(handle);
    wrap.appendChild(sc);
    wrap.appendChild(bar);
    var b = { sc: sc, bar: bar, handle: handle };
    bars.push(b);
    onDragStart(handle, function (ev) {
      dragging = { b: b, y: pointOf(ev).clientY, top: sc.scrollTop };
      ev.stopPropagation();
    });
    bar.addEventListener("mousedown", function (ev) {
      var r = handle.getBoundingClientRect();
      sc.scrollTop += ev.clientY < r.top ? -sc.clientHeight : sc.clientHeight;
      updateBar(b);
    });
    sc.addEventListener("scroll", function () {
      updateBar(b);
    });
    return wrap;
  }

  var dragging = null; // {b, y, top}: the scrollbar whose handle is held

  function dragBar(ev) {
    if (!dragging) return;
    var b = dragging.b;
    var room = b.bar.clientHeight - b.handle.offsetHeight;
    if (room > 0) b.sc.scrollTop = dragging.top + ((pointOf(ev).clientY - dragging.y) * (b.sc.scrollHeight - b.sc.clientHeight)) / room;
    updateBar(b);
  }

  // updateBar sizes the handle to the visible part of the content (no smaller than its CSS min-height)
  // and moves it to the scroll position.
  function updateBar(b) {
    var view = b.sc.clientHeight, full = b.sc.scrollHeight;
    if (full <= view + 1) {
      b.bar.style.display = "none";
      return;
    }
    b.bar.style.display = "flex";
    var track = b.bar.clientHeight;
    b.handle.style.height = Math.round((track * view) / full) + "px";
    var h = b.handle.offsetHeight;
    b.handle.style.top = Math.round(((track - h) * b.sc.scrollTop) / (full - view)) + "px";
  }

  // canvas is a view on a plane of absolutely placed children that the player drags around and zooms
  // (the mouse wheel, two fingers, or the + and - buttons at its bottom right; the zoom stays at the
  // pointer or between the fingers). Where it was dragged to and the zoom are kept by its key, so a
  // screen drawn again (a quest opened) stays as it was; the first time the point (focusX, focusY) in
  // rem is put in the middle, at the zoom last used. The plane starts zoomed in a step on a touch
  // screen, where the nodes are pressed with a finger.
  var pans = {};
  var ZOOM_MIN = 0.5;
  var ZOOM_MAX = 3;
  var ZOOM_STEP = 1.25;
  var lastZoom = 0; // the zoom of the canvas last zoomed, 0 before any

  function buildCanvas(n, rec) {
    var view = el("div", "oui-canvas");
    var plane = el("div", "oui-canvas-plane");
    view.appendChild(plane);
    appendChildren(plane, n.children, rec);
    var pan = pans[n.key];
    function place() {
      plane.style.left = pan.x + "px";
      plane.style.top = pan.y + "px";
      plane.style.transform = "scale(" + pan.z + ")";
    }
    // zoomTo zooms to z keeping the point (cx, cy) of the view, in px, where it is (the middle when
    // none is given).
    function zoomTo(z, cx, cy) {
      if (!pan) return;
      z = Math.max(ZOOM_MIN, Math.min(ZOOM_MAX, z));
      if (cx === undefined) {
        cx = view.clientWidth / 2;
        cy = view.clientHeight / 2;
      }
      pan.x = cx - ((cx - pan.x) * z) / pan.z;
      pan.y = cy - ((cy - pan.y) * z) / pan.z;
      pan.z = lastZoom = z;
      place();
    }
    // inView is where a client point is in the view.
    function inView(x, y) {
      var r = view.getBoundingClientRect();
      return { x: x - r.left - (view.clientLeft || 0), y: y - r.top - (view.clientTop || 0) };
    }
    setTimeout(function () {
      if (!pan) {
        var rem = parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
        var z = lastZoom || (touchSeen ? ZOOM_STEP : 1);
        pan = pans[n.key] = { x: view.clientWidth / 2 - (n.focusX || 0) * rem * z, y: view.clientHeight / 2 - (n.focusY || 0) * rem * z, z: z };
      }
      place();
    }, 0);
    onDragStart(view, function (ev) {
      if (ev.touches && ev.touches.length >= 2) return;
      if (ev.target !== view && ev.target !== plane && !/oui-line-seg|oui-canvas-deco/.test(ev.target.className)) return;
      if (!pan) return;
      if (tip) tip.style.display = "none";
      var at = pointOf(ev);
      panning = { pan: pan, place: place, x: at.clientX, y: at.clientY, px: pan.x, py: pan.y };
    });
    view.addEventListener("wheel", function (ev) {
      if (!pan || !ev.deltaY) return;
      var at = inView(ev.clientX, ev.clientY);
      zoomTo(ev.deltaY < 0 ? pan.z * ZOOM_STEP : pan.z / ZOOM_STEP, at.x, at.y);
      if (ev.preventDefault) ev.preventDefault();
    });
    // A second finger, wherever it lands in the view, turns the drag into a pinch.
    view.addEventListener("touchstart", function (ev) {
      if (!pan || !ev.touches || ev.touches.length < 2) return;
      panning = null;
      if (tip) tip.style.display = "none";
      pinching = { zoomTo: zoomTo, inView: inView, d: fingerGap(ev.touches) || 1, z: pan.z };
    });
    var zoom = el("div", "oui-canvas-zoom");
    [["+", ZOOM_STEP], ["-", 1 / ZOOM_STEP]].forEach(function (b) {
      var button = el("div", "oui-canvas-zoom-button", b[0]);
      button.addEventListener("mousedown", function (ev) {
        if (ev.button !== undefined && ev.button !== 0) return;
        claimed = ev;
        if (pan) zoomTo(pan.z * b[1]);
      });
      zoom.appendChild(button);
    });
    view.appendChild(zoom);
    return view;
  }

  var panning = null; // {pan, place, x, y, px, py}: the canvas being dragged
  var pinching = null; // {zoomTo, inView, d, z}: the canvas being pinched, from the finger gap d at zoom z

  function fingerGap(touches) {
    return Math.sqrt(Math.pow(touches[1].clientX - touches[0].clientX, 2) + Math.pow(touches[1].clientY - touches[0].clientY, 2));
  }

  function dragCanvas(ev) {
    if (pinching) {
      if (!ev.touches || ev.touches.length < 2) return;
      var mid = pinching.inView((ev.touches[0].clientX + ev.touches[1].clientX) / 2, (ev.touches[0].clientY + ev.touches[1].clientY) / 2);
      pinching.zoomTo((pinching.z * fingerGap(ev.touches)) / pinching.d, mid.x, mid.y);
      return;
    }
    if (!panning) return;
    var at = pointOf(ev);
    panning.pan.x = panning.px + at.clientX - panning.x;
    panning.pan.y = panning.py + at.clientY - panning.y;
    panning.place();
  }

  // layoutPlanes places the children of every plane node on the screen: a straight-down camera at
  // height dist with the vertical field of view fov shows dist*tan(fov/2) blocks from the middle of the
  // screen to its top edge, so a block is that many pixels (the same across, as pixels are square).
  function layoutPlanes() {
    if (!root) return;
    var planes = root.querySelectorAll(".oui-plane");
    for (var i = 0; i < planes.length; i++) {
      var rec = planes[i].__rec;
      if (!rec) continue;
      var n = rec.n;
      var half = (n.dist || 1) * Math.tan((((n.fov || 60) / 2) * Math.PI) / 180);
      var scale = window.innerHeight / 2 / half;
      var cx = window.innerWidth / 2, cy = window.innerHeight / 2;
      rec.kids.forEach(function (k) {
        var at = k.n.at;
        if (!at) return;
        var st = k.el.style;
        st.left = Math.round(cx + at[0] * scale) + "px";
        st.top = Math.round(cy + at[1] * scale) + "px";
        st.width = Math.round(at[2] * scale) + "px";
        st.height = Math.round(at[3] * scale) + "px";
      });
    }
  }

  function appendChildren(e, children, rec) {
    rec.box = e;
    (children || []).forEach(function (c) {
      var r = make(c);
      rec.kids.push(r);
      e.appendChild(r.el);
    });
  }

  // gridCell is the cell of a grid that holds child.
  function gridCell(n, child) {
    var cell = el("div", "oui-cell");
    cell.style.width = 100 / (n.cols || 10) + "%";
    cell.appendChild(child);
    return cell;
  }

  function isScroll(n) {
    return /(^| )mc-scroll( |$)/.test(n["class"] || "");
  }

  // --- patching --------------------------------------------------------------------------------

  // id and tip are read from the record when used; children are patched one by one.
  var SOFT = { id: true, tip: true, children: true };

  // Keys of a node a record can take in place, by node type, besides SOFT.
  var IN_PLACE = {
    col: {}, row: {}, grid: {}, region: {}, frame: {}, arrow: {}, space: {},
    box: { "class": true, style: true, on: true, at: true },
    text: { "class": true, style: true, text: true },
    img: { "class": true, style: true, icon: true, size: true },
    canvas: { focusX: true, focusY: true }, // only the first drawing reads the focus
    plane: { fov: true, dist: true }
  };

  // Node types whose presses read the id from the record (the others keep the id they were built with).
  var LIVE_ID = { box: true, button: true, col: true, row: true, grid: true, text: true, img: true, region: true, frame: true, arrow: true, space: true, canvas: true, plane: true };

  function same(a, b) {
    if (a === b) return true;
    if (typeof a !== "object" || typeof b !== "object" || a === null || b === null) return false;
    return JSON.stringify(a) === JSON.stringify(b);
  }

  // changedKeys are the keys of a and b, not SOFT, whose values differ.
  function changedKeys(a, b) {
    var keys = {}, out = [], k;
    for (k in a) keys[k] = true;
    for (k in b) keys[k] = true;
    for (k in keys) if (!SOFT[k] && !same(a[k], b[k])) out.push(k);
    return out;
  }

  // takesInPlace reports whether rec can become node n without being built again.
  function takesInPlace(rec, n, nInk) {
    var o = rec.n;
    if (o.t !== n.t || rec.ink !== nInk) return false;
    var allowed = IN_PLACE[n.t];
    if (!allowed) return changedKeys(o, n).length === 0 && (LIVE_ID[n.t] || same(o.id, n.id));
    if (!LIVE_ID[n.t] && !same(o.id, n.id)) return false;
    // A binding or a scroll wrapper is set up when the node is built.
    if (o.bind || n.bind || o.activeBind || n.activeBind) {
      if (changedKeys(o, n).length) return false;
    }
    if (isScroll(o) !== isScroll(n)) return false;
    var keys = changedKeys(o, n);
    for (var i = 0; i < keys.length; i++) if (!allowed[keys[i]]) return false;
    return true;
  }

  // patch makes the drawing of rec show n, in place when it can, and returns the record that shows it
  // (rec, or a new one the caller puts in rec's place). parentInk is the ink around n.
  function patch(rec, n, parentInk) {
    var nInk = n.ink || parentInk;
    if (!takesInPlace(rec, n, nInk)) {
      var outer = ink;
      ink = parentInk;
      try {
        return make(n);
      } finally {
        ink = outer;
      }
    }
    var o = rec.n, e = rec.inner;
    rec.n = n;
    if (!same(o["class"], n["class"]) || !same(o.on, n.on)) {
      var base = n.t === "img" ? "oui-img" : "oui-" + n.t;
      if (n.t === "box" && n.on) base += " on";
      e.className = base + (n["class"] ? " " + n["class"] : "");
    }
    if (!same(o.style, n.style)) {
      var k;
      for (k in o.style || {}) if (!n.style || !(k in n.style)) e.style[k] = "";
      for (k in n.style || {}) e.style[k] = n.style[k];
    }
    if (n.t === "text" && o.text !== n.text) rich(e, n.text, rec.ink === "panel");
    if (n.t === "img" && (o.icon !== n.icon || o.size !== n.size)) e.src = iconSrc(n.icon, n.size || 64);
    if (n.tip && !rec.tipped) withTip(e, rec);
    if (tipOwner === e && tip) {
      if (n.tip) rich(tip, n.tip, false);
      else tip.style.display = "none";
    }
    if (rec.box) patchKids(rec, n.children || []);
    return rec;
  }

  // patchKids makes the children of rec those of the list: each child patched in its place, the new
  // ones added at the end, the ones left over removed.
  function patchKids(rec, children) {
    var kids = rec.kids, out = [], i;
    for (i = 0; i < children.length; i++) {
      if (i < kids.length) {
        var r = patch(kids[i], children[i], rec.ink);
        if (r !== kids[i]) kids[i].el.parentNode.replaceChild(r.el, kids[i].el);
        out.push(r);
        continue;
      }
      var outer = ink;
      ink = rec.ink;
      try {
        var added = make(children[i]);
      } finally {
        ink = outer;
      }
      rec.box.appendChild(rec.grid ? gridCell(rec.n, added.el) : added.el);
      out.push(added);
    }
    for (; i < kids.length; i++) {
      var gone = rec.grid ? kids[i].el.parentNode : kids[i].el;
      gone.parentNode.removeChild(gone);
    }
    rec.kids = out;
  }

  // prune drops the bindings, scrollbars and tooltip of what a patch took off the screen.
  function prune() {
    if (!root) return;
    bound = bound.filter(function (b) {
      return root.contains(b.el);
    });
    bars = bars.filter(function (b) {
      return root.contains(b.sc);
    });
    inputs = inputs.filter(function (i) {
      return root.contains(i.el);
    });
    pressables = pressables.filter(function (p) {
      return root.contains(p.e);
    });
    if (tip && tipOwner && !root.contains(tipOwner)) hideTip();
    if (armed && !root.contains(armed)) armed = null;
    keepFocus();
  }

  // fillRegion draws a region from its variable, patching what it showed before.
  function fillRegion(name) {
    var e = regions[name];
    if (!e) return;
    var n = varOr("@" + name, null);
    var old = regionRecs[name];
    var rInk = regionInk[name] || ink;
    if (!n) {
      e.innerHTML = "";
      delete regionRecs[name];
      return;
    }
    var outer = building;
    building = name;
    try {
      var r;
      if (old && old.el.parentNode === e) {
        r = patch(old, n, rInk);
        if (r !== old) e.replaceChild(r.el, old.el);
      } else {
        e.innerHTML = "";
        var outerInk = ink;
        ink = rInk;
        try {
          r = make(n);
        } finally {
          ink = outerInk;
        }
        e.appendChild(r.el);
      }
      regionRecs[name] = r;
    } finally {
      building = outer;
    }
  }

  // size 0 leaves the size to the CSS (the mc-slot classes); the icon is then asked at 64px.
  function buildItem(n) {
    var size = n.size === 0 ? 0 : n.size || 48;
    var box = el("div", "oui-item");
    var img = el("img", "oui-item-img");
    if (size) {
      img.style.width = size + "px";
      img.style.height = size + "px";
    }
    img.src = iconSrc(varOr(n.bind, n.icon || n.id), size || 64);
    if (n.bind) bind({ el: img, key: n.bind, kind: "item", size: size });
    if (n.activeBind) {
      if (varOr(n.activeBind, false)) box.className += " on";
      bind({ el: box, key: n.activeBind, kind: "on" });
    }
    box.appendChild(img);
    if (n.sel) box.className += " on";
    if (n.count !== undefined && n.count !== null && n.count !== "") {
      var count = el("div", n.countClass || "oui-item-count");
      rich(count, n.count, false);
      box.appendChild(count);
    }
    if (n.text) {
      var label = el("div", "oui-item-label");
      rich(label, n.text);
      box.appendChild(label);
    }
    if (n.id && n.clickable !== false) {
      box.className += " pressable";
      onPress(
        box,
        function () {
          send(n.id, "click", null);
        },
        function () {
          return n;
        }
      );
    }
    return box;
  }

  // A container slot: the item comes from a variable {id, n} (empty or missing means an empty slot), so
  // the server can move items by sending only the slots that changed.
  function showSlot(b, v) {
    var id = v && v.id ? v.id : "";
    if (id !== b.id) {
      b.id = id;
      if (id) b.img.src = iconSrc(id, b.size);
      b.img.style.visibility = id ? "visible" : "hidden";
    }
    b.count.textContent = v && v.n > 1 ? String(v.n) : "";
  }

  function buildSlot(n) {
    var size = n.size || 40;
    var box = el("div", "oui-slot");
    var img = el("img", "oui-slot-img");
    img.style.width = size + "px";
    img.style.height = size + "px";
    var count = el("div", "oui-slot-count");
    box.appendChild(img);
    box.appendChild(count);
    var b = { el: box, key: n.bind, kind: "slot", size: size, img: img, count: count, id: null };
    showSlot(b, varOr(n.bind, null));
    if (n.bind) bind(b);
    if (n.raw) rawSlot(box, n);
    else
      onPress(box, function () {
        send(n.id, "click", null);
      });
    return box;
  }

  // A raw slot reports every press with its button and modifier keys ("slot"), the slots a held button
  // passed over when it is let go ("slotup"), and the keys pressed while the pointer is over it
  // ("slotkey"), so the server can play the vanilla inventory: right button, shift, double click,
  // splitting by dragging, number keys.
  var DOUBLE_MS = 300;
  var rawDown = null; // {id, b, slots}: the press of a raw slot whose button is still held
  var lastRaw = null; // {id, b, at}: the last press of a raw slot, to tell a double click
  var hoverSlot = null; // the id of the raw slot under the pointer

  function rawSlot(box, n) {
    box.addEventListener("mousedown", function (ev) {
      if (claimed === ev) return;
      claimed = ev;
      var b = ev.button || 0, at = Date.now();
      var dbl = !!lastRaw && lastRaw.id === n.id && lastRaw.b === b && at - lastRaw.at < DOUBLE_MS;
      lastRaw = { id: n.id, b: b, at: at };
      rawDown = { id: n.id, b: b, slots: [n.id] };
      send(n.id, "slot", { b: b, shift: !!ev.shiftKey, ctrl: !!ev.ctrlKey, alt: !!ev.altKey, dbl: dbl });
    });
    box.addEventListener("mouseover", function () {
      hoverSlot = n.id;
      if (rawDown && rawDown.slots.indexOf(n.id) < 0) rawDown.slots.push(n.id);
    });
    box.addEventListener("mouseout", function () {
      if (hoverSlot === n.id) hoverSlot = null;
    });
  }

  function releaseRaw() {
    if (!rawDown) return;
    var d = rawDown;
    rawDown = null;
    send(d.id, "slotup", { b: d.b, slots: d.slots });
  }

  // The cursor node is the item held on the pointer: a slot variable drawn where the pointer is. It sits
  // in a box at the top left of the screen (mc-screen), so the pointer's client position places it.
  var pointer = { x: 0, y: 0 };
  var cursors = [];

  function buildCursor(n) {
    var size = n.size || 40;
    var box = el("div", "oui-cursor");
    var img = el("img", "oui-slot-img");
    img.style.width = size + "px";
    img.style.height = size + "px";
    var count = el("div", "oui-slot-count");
    box.appendChild(img);
    box.appendChild(count);
    var b = { el: box, key: n.bind, kind: "slot", size: size, img: img, count: count, id: null };
    showSlot(b, varOr(n.bind, null));
    if (n.bind) bind(b);
    var c = { el: box, size: size };
    cursors.push(c);
    placeCursor(c);
    return box;
  }

  function placeCursor(c) {
    c.el.style.left = Math.round(pointer.x - c.size / 2) + "px";
    c.el.style.top = Math.round(pointer.y - c.size / 2) + "px";
  }

  function trackPointer(ev) {
    var at = pointOf(ev);
    if (at.clientX === undefined) return;
    pointer = { x: at.clientX, y: at.clientY };
    if (!cursors.length) return;
    cursors = cursors.filter(function (c) {
      return root && root.contains(c.el);
    });
    cursors.forEach(placeCursor);
  }

  // The probe node lists what the page knows: the engine command groups and their commands, and the
  // last input events with their buttons, keys and modifiers. Every event but a move is sent to the
  // server too ("probe"), so the server log keeps what a device delivers. The document's events are
  // taken in the capture phase, so focus, blur and scroll, which do not bubble, come too; the window's
  // own events (PROBE_WINDOW_TYPES) are taken on the window. Gamepads are read with
  // navigator.getGamepads on the timer (probeGamepads).
  var PROBE_TYPES = ["mousedown", "mouseup", "click", "dblclick", "contextmenu", "wheel", "keydown", "keyup", "keypress",
    "touchstart", "touchend", "touchcancel", "mousemove", "touchmove", "pointerdown", "pointerup", "pointercancel", "pointermove",
    "focus", "blur", "focusin", "focusout", "input", "change", "compositionstart", "compositionupdate", "compositionend",
    "scroll", "visibilitychange"];
  var PROBE_WINDOW_TYPES = ["resize", "gamepadconnected", "gamepaddisconnected"];
  var PROBE_MOVES = { mousemove: true, touchmove: true, pointermove: true, scroll: true };
  var probeEl = null;
  var probeLog = [];
  var probeMoves = { mousemove: 0, touchmove: 0, pointermove: 0, scroll: 0 };
  var probeDownAt = 0;
  var probePads = "";
  var envSent = false;

  function commandList() {
    var c = window.__commands__ || {}, out = [];
    for (var g in c) {
      var names = [];
      for (var k in c[g]) names.push(k);
      out.push(g + ": " + names.join(" "));
    }
    return out;
  }

  function probeLive() {
    return probeEl && root && root.contains(probeEl);
  }

  function probeEvent(ev) {
    if (!probeLive()) return;
    var t = ev.type;
    if (PROBE_MOVES[t]) {
      probeMoves[t]++;
      return drawProbe();
    }
    var parts = [t];
    if (/mouse|click|contextmenu|wheel|pointer/.test(t)) parts.push("button=" + ev.button, "buttons=" + ev.buttons, "detail=" + ev.detail);
    if (/down$|start$/.test(t) && ev.clientX !== undefined) parts.push("x=" + Math.round(ev.clientX), "y=" + Math.round(ev.clientY));
    if (/^pointer/.test(t)) parts.push("pointerType=" + ev.pointerType, "pointerId=" + ev.pointerId);
    if (t === "wheel") parts.push("dx=" + Math.round(ev.deltaX || 0), "dy=" + Math.round(ev.deltaY || 0), "mode=" + ev.deltaMode);
    if (/^key/.test(t)) parts.push("key=" + ev.key, "code=" + ev.code, "keyCode=" + ev.keyCode, "repeat=" + ev.repeat);
    if (/^composition|^input$/.test(t)) parts.push("data=" + ev.data);
    if (t === "change" || t === "input") parts.push("value=" + (ev.target && ev.target.value));
    if (t === "resize") parts.push("w=" + window.innerWidth, "h=" + window.innerHeight);
    if (/^gamepad/.test(t) && ev.gamepad) parts.push("index=" + ev.gamepad.index, "id=" + ev.gamepad.id);
    if (ev.touches) parts.push("touches=" + ev.touches.length);
    if (t === "mousedown" || t === "touchstart") probeDownAt = Date.now();
    if ((t === "mouseup" || t === "touchend") && probeDownAt) parts.push("held=" + (Date.now() - probeDownAt) + "ms");
    var mods = [];
    if (ev.shiftKey) mods.push("shift");
    if (ev.ctrlKey) mods.push("ctrl");
    if (ev.altKey) mods.push("alt");
    if (ev.metaKey) mods.push("meta");
    if (mods.length) parts.push(mods.join("+"));
    if (ev.target && ev.target.className) parts.push("on=" + String(ev.target.className).split(" ")[0]);
    probeLine(parts.join(" "));
  }

  // A line like the last one (a held key's repeats) counts on the last line and is not sent again, so
  // one key does not push the other events off the list.
  var probeLast = "", probeRepeats = 0;
  function probeLine(line) {
    if (line === probeLast) {
      probeRepeats++;
      probeLog[0] = line + " §7x" + (probeRepeats + 1);
      return drawProbe();
    }
    probeLast = line;
    probeRepeats = 0;
    probeLog.unshift(line);
    if (probeLog.length > 16) probeLog.length = 16;
    drawProbe();
    send("probe", "probe", line);
  }

  // probeGamepads logs a gamepad's pressed buttons and moved axes whenever they change.
  function probeGamepads() {
    if (!navigator.getGamepads) return;
    var pads;
    try {
      pads = navigator.getGamepads() || [];
    } catch (e) {
      return;
    }
    var out = [];
    for (var i = 0; i < pads.length; i++) {
      var g = pads[i];
      if (!g) continue;
      var held = [], axes = [];
      (g.buttons || []).forEach(function (b, j) {
        if (b && (b.pressed || b.value > 0.5)) held.push(j);
      });
      (g.axes || []).forEach(function (a, j) {
        if (Math.abs(a) > 0.3) axes.push(j + ":" + a.toFixed(1));
      });
      out.push("gamepad " + g.index + " buttons=" + held.join(",") + " axes=" + axes.join(","));
    }
    var s = out.join(" | ");
    if (s === probePads) return;
    probePads = s;
    if (s) probeLine(s);
  }

  function drawProbe() {
    if (!probeEl) return;
    var cmds = commandList();
    if (!envSent && probeLive()) {
      envSent = true;
      send("env", "probe", "engine=" + typeof engine + " size=" + window.innerWidth + "x" + window.innerHeight + " getGamepads=" +
        typeof navigator.getGamepads + " PointerEvent=" + typeof window.PointerEvent + " href=" + location.href + " | " + cmds.join(" | "));
    }
    var lines = ["§emoves§r mouse " + probeMoves.mousemove + ", touch " + probeMoves.touchmove + ", pointer " + probeMoves.pointermove +
      ", scroll " + probeMoves.scroll, "§eevents§r"].concat(probeLog);
    lines.push("§ecommand groups§r " + cmds.length);
    rich(probeEl, lines.concat(cmds).join("\n"), false);
  }

  // The restest node tries, one at a time, the ways the server could give the page JS, CSS and HTML:
  // the text in the node (js, css, html: sent in the data store) run through eval, Function and inline,
  // data: and blob: elements; the pack's files (/hbui/oui-probe.*) through elements, fetch and
  // XMLHttpRequest; and the same kinds of files from each HTTP server of `http` ([{name, base}]). A
  // script passes when it calls __ouiRes, a style sheet when it makes .oui-res-probe 13px wide, an HTML
  // text when it holds oui-res-html. Every result is drawn and sent to the server ("res").
  var RES_TIMEOUT = 4000;
  var resRan = null;
  window.__ouiRes = function (tag) {
    resRan = String(tag);
  };

  function resEnv() {
    return "href=" + location.href + " origin=" + location.origin + " fetch=" + typeof window.fetch + " XMLHttpRequest=" +
      typeof window.XMLHttpRequest + " Blob=" + typeof window.Blob + " createObjectURL=" + typeof (window.URL && URL.createObjectURL);
  }

  // Gameface lays the page out on its next frame, so a style sheet is judged on an element that stays
  // in the page (off the screen) some time after the sheet came (RES_SETTLE_MS).
  var RES_SETTLE_MS = 300;
  var resProbe = null;

  function resStyled() {
    if (!resProbe) {
      resProbe = el("div", "oui-res-probe");
      resProbe.style.position = "absolute";
      resProbe.style.left = "-100px";
      resProbe.style.top = "0px";
      document.body.appendChild(resProbe);
    }
    return Math.round(resProbe.getBoundingClientRect().width) === 13;
  }

  function resStyleNote() {
    if (!resProbe) return "";
    var cs = "?";
    try {
      cs = window.getComputedStyle(resProbe).width;
    } catch (e) {}
    return " w=" + resProbe.getBoundingClientRect().width + " computed=" + cs;
  }

  function resHTML(text) {
    return typeof text === "string" && text.indexOf("oui-res-html") >= 0;
  }

  // resElement adds e to the head and passes the test when ok() holds RES_SETTLE_MS after it loads (or
  // after it is added, when it has no load to wait for), then takes e away again.
  function resElement(e, ok, now, note) {
    return {
      check: ok,
      run: function (finish) {
        var over = false;
        function done(label) {
          if (over) return;
          over = true;
          setTimeout(function () {
            var pass = ok();
            var detail = label + (note ? note() : "");
            if (e.parentNode) e.parentNode.removeChild(e);
            finish(pass, detail);
          }, RES_SETTLE_MS);
        }
        e.onload = function () {
          done("onload");
        };
        e.onerror = function () {
          if (over) return;
          over = true;
          if (e.parentNode) e.parentNode.removeChild(e);
          finish(false, "onerror");
        };
        resStyled();
        document.head.appendChild(e);
        if (now) done("added");
      }
    };
  }

  function resScript(src, text) {
    var s = document.createElement("script");
    if (src) s.src = src;
    else s.textContent = text;
    return resElement(s, function () {
      return resRan !== null;
    }, !src);
  }

  function resSheet(href) {
    var l = document.createElement("link");
    l.rel = "stylesheet";
    l.href = href;
    return resElement(l, resStyled, false, resStyleNote);
  }

  function resImage(src) {
    var i = el("img");
    return {
      run: function (finish) {
        i.onload = function () {
          finish(true, "onload");
        };
        i.onerror = function () {
          finish(false, "onerror");
        };
        i.src = src;
      }
    };
  }

  function resFetch(url) {
    return {
      run: function (finish) {
        if (typeof window.fetch !== "function") return finish(false, "no fetch");
        window
          .fetch(url)
          .then(function (r) {
            return r.text().then(function (t) {
              finish(resHTML(t), "status=" + r.status + " " + t.length + "B");
            });
          })
          .catch(function (err) {
            finish(false, "catch " + err);
          });
      }
    };
  }

  function resXHR(url) {
    return {
      run: function (finish) {
        if (typeof window.XMLHttpRequest === "undefined") return finish(false, "no XMLHttpRequest");
        var x = new XMLHttpRequest();
        x.onload = function () {
          finish(resHTML(x.responseText), "status=" + x.status + " " + String(x.responseText || "").length + "B");
        };
        x.onerror = function () {
          finish(false, "onerror status=" + x.status);
        };
        x.open("GET", url);
        x.send();
      }
    };
  }

  function resTests(n) {
    var js = n.js || "", css = n.css || "", html = n.html || "";
    function now(f) {
      return {
        run: function (finish) {
          f(finish);
        }
      };
    }
    var tests = [
      ["packet eval", now(function (finish) {
        (0, eval)(js);
        finish(resRan !== null);
      })],
      ["packet Function", now(function (finish) {
        new Function(js)();
        finish(resRan !== null);
      })],
      ["packet inline script", resScript(null, js)],
      ["packet data: script", resScript("data:text/javascript;charset=utf-8," + encodeURIComponent(js))],
      ["packet blob: script", now(function (finish) {
        if (typeof window.Blob === "undefined" || !window.URL || !URL.createObjectURL) return finish(false, "no Blob");
        resScript(URL.createObjectURL(new Blob([js]))).run(finish);
      })],
      ["packet style", resElement(el("style", null, css), resStyled, true, resStyleNote)],
      ["packet data: link", resSheet("data:text/css;charset=utf-8," + encodeURIComponent(css))],
      ["packet innerHTML", now(function (finish) {
        var d = el("div");
        d.innerHTML = html;
        finish(!!d.querySelector(".oui-res-html"));
      })],
      ["pack script", resScript("/hbui/oui-probe.js")],
      ["pack link", resSheet("/hbui/oui-probe.css")],
      ["pack fetch", resFetch("/hbui/oui-probe.html")],
      ["pack xhr", resXHR("/hbui/oui-probe.html")],
      ["pack:// xhr", resXHR("pack://hbui/oui-probe.html")]
    ];
    (n.http || []).forEach(function (h) {
      var q = "?t=" + encodeURIComponent(h.name);
      tests.push(
        [h.name + " script", resScript(h.base + "/probe.js" + q)],
        [h.name + " link", resSheet(h.base + "/probe.css" + q)],
        [h.name + " img", resImage(h.base + "/probe.png" + q)],
        [h.name + " fetch", resFetch(h.base + "/probe.html" + q)],
        [h.name + " xhr", resXHR(h.base + "/probe.html" + q)]
      );
    });
    return tests;
  }

  function buildResTest(n) {
    var e = el("div", "oui-probe");
    var lines = ["§eenv§r " + resEnv()];
    function line(s) {
      lines.push(s);
      rich(e, lines.join("\n"), false);
    }
    rich(e, lines[0], false);
    send("res-env", "probe", resEnv());
    var tests = resTests(n), i = 0;
    function next() {
      if (!root || !root.contains(e)) return;
      if (i >= tests.length) return line("§edone§r");
      var name = tests[i][0], t = tests[i][1], over = false;
      i++;
      resRan = null;
      var timer = setTimeout(function () {
        finish(t.check ? t.check() : false, "timeout");
      }, RES_TIMEOUT);
      function finish(ok, detail) {
        if (over) return;
        over = true;
        clearTimeout(timer);
        var s = name + " " + (ok ? "ok" : "fail") + (detail ? " " + detail : "") + (resRan !== null ? " ran=" + resRan : "");
        line((ok ? "§a" : "§c") + s);
        send("res", "probe", s);
        setTimeout(next, 50);
      }
      try {
        t.run(finish);
      } catch (err) {
        finish(false, "throw " + err);
      }
    }
    setTimeout(next, 100);
    return e;
  }

  // The engine node reads the vanilla container queries (vanillaGameplayContainerItemQuery for
  // [container, index], vanillaGameplayContainerSizeQuery for a container: the ones the vanilla Ore UI
  // slots read) and calls the vanilla container commands (vanillaGameplayContainerAPICommands) with the
  // arguments the server gives, to see whether they reach the player's inventory outside a container
  // screen. A call with cmd "__report" sends what the queries hold.
  var engineVals = {};
  var engineSubs = {};
  var engineEls = [];

  function engineWatch(key, name, params) {
    if (engineSubs[key]) return;
    engineSubs[key] = true;
    engineVals[key] = "(none)";
    subscribe(name, params, function (v) {
      v = unwrap(v);
      engineVals[key] = v && typeof v === "object" ? JSON.stringify(v) : String(v);
      drawEngines();
    });
  }

  function buildEngine(n) {
    var e = el("div", "oui-engine");
    var list = el("div", "oui-engine-list");
    var result = el("div", "oui-engine-result");
    var calls = el("div", "oui-row oui-engine-calls");
    (n.sizes || []).forEach(function (c) {
      engineWatch("size " + c, "vanillaGameplayContainerSizeQuery", [c]);
    });
    (n.slots || []).forEach(function (s) {
      engineWatch(s[0] + ":" + s[1], "vanillaGameplayContainerItemQuery", [s[0], s[1]]);
    });
    (n.calls || []).forEach(function (c) {
      var b = el("div", "oui-button", c.label);
      onPress(b, function () {
        var out;
        if (c.cmd === "__report") out = JSON.stringify(engineVals);
        else {
          var f = command("vanillaGameplayContainerAPICommands", c.cmd);
          if (!f) out = "missing";
          else {
            try {
              out = "ok " + JSON.stringify(f.apply(null, c.args || []));
            } catch (err) {
              out = "throw " + err;
            }
          }
        }
        rich(result, c.label + ": " + out, false);
        send(n.id || "engine", "engine", c.label + ": " + out);
      });
      calls.appendChild(b);
    });
    e.appendChild(calls);
    e.appendChild(result);
    e.appendChild(list);
    var keys = (n.sizes || []).map(function (c) {
      return "size " + c;
    });
    (n.slots || []).forEach(function (s) {
      keys.push(s[0] + ":" + s[1]);
    });
    var x = { el: e, list: list, keys: keys, labels: n.labels || {} };
    engineEls.push(x);
    drawEngine(x);
    return e;
  }

  function drawEngine(x) {
    rich(x.list, x.keys.map(function (k) {
      return (x.labels[k] ? x.labels[k] + " " : "") + k + " = " + engineVals[k];
    }).join("\n"), false);
  }

  function drawEngines() {
    engineEls = engineEls.filter(function (x) {
      return root && root.contains(x.el);
    });
    engineEls.forEach(drawEngine);
  }

  function labelled(n, control) {
    var row = el("div", "oui-row oui-field");
    var label = el("div", "oui-label");
    rich(label, n.text);
    row.appendChild(label);
    row.appendChild(control);
    return row;
  }

  function buildToggle(n) {
    var on = !!n.value;
    var sw = el("div", "oui-switch" + (on ? " on" : ""));
    sw.appendChild(el("div", "oui-knob"));
    onPress(sw, function () {
      on = !on;
      sw.className = "oui-switch" + (on ? " on" : "");
      send(n.id, "change", on);
    });
    return labelled(n, sw);
  }

  function buildSlider(n) {
    var min = n.min || 0, max = n.max === undefined ? 100 : n.max, step = n.step || 1;
    var value = n.value === undefined ? min : n.value;
    var track = el("div", "oui-track");
    var fill = el("div", "oui-track-fill");
    var label = el("div", "oui-value");
    track.appendChild(fill);
    function show() {
      fill.style.width = ((value - min) / (max - min)) * 100 + "%";
      label.textContent = String(value);
    }
    function at(ev) {
      var r = track.getBoundingClientRect();
      var f = Math.min(1, Math.max(0, (ev.clientX - r.left) / r.width));
      value = Math.round((min + f * (max - min)) / step) * step;
      show();
    }
    var dragging = false;
    track.addEventListener("mousedown", function (ev) {
      dragging = true;
      at(ev);
    });
    document.addEventListener("mousemove", function (ev) {
      if (dragging) at(ev);
    });
    document.addEventListener("mouseup", function () {
      if (!dragging) return;
      dragging = false;
      send(n.id, "change", value);
    });
    show();
    var wrap = el("div", "oui-row");
    wrap.appendChild(track);
    wrap.appendChild(label);
    return labelled(n, wrap);
  }

  // inputs are the input boxes drawn: {id, el}.
  var inputs = [];

  function inputValues() {
    var out = null;
    inputs.forEach(function (i) {
      if (!i.id || !root || !root.contains(i.el)) return;
      out = out || {};
      out[i.id] = i.el.value;
    });
    return out;
  }

  // An input box sends its text a moment after the typing stops, or at Enter; with `enter` (the id of a
  // button) Enter presses that button instead.
  function buildInput(n) {
    var input = el("input", "oui-input");
    input.type = "text";
    input.value = n.value || "";
    if (n.placeholder) input.placeholder = localize(n.placeholder);
    inputs.push({ id: n.id, el: input });
    var timer = null;
    function flush() {
      if (timer) clearTimeout(timer);
      timer = null;
      send(n.id, "change", input.value);
    }
    input.addEventListener("input", function () {
      if (timer) clearTimeout(timer);
      timer = setTimeout(flush, n.delay || 400);
    });
    input.addEventListener("keydown", function (ev) {
      if (ev.keyCode !== 13) return;
      if (n.enter) {
        if (timer) clearTimeout(timer);
        timer = null;
        send(n.enter, "click", null);
      } else flush();
    });
    return n.text ? labelled(n, input) : input;
  }

  function buildSelect(n) {
    var row = el("div", "oui-row oui-options");
    var buttons = [];
    (n.options || []).forEach(function (opt) {
      var b = el("div", "oui-button" + (opt === n.value ? " on" : ""), localize(opt));
      onPress(b, function () {
        buttons.forEach(function (x) {
          x.className = "oui-button";
        });
        b.className = "oui-button on";
        send(n.id, "change", opt);
      });
      buttons.push(b);
      row.appendChild(b);
    });
    return labelled(n, row);
  }

  function applyVars(changed) {
    for (var k in changed) {
      if (k.charAt(0) === "@") fillRegion(k.slice(1));
    }
    prune();
    bound.forEach(function (b) {
      if (!Object.prototype.hasOwnProperty.call(changed, b.key)) return;
      var v = changed[b.key];
      if (b.kind === "text") rich(b.el, v, b.panel);
      else if (b.kind === "width") b.el.style.width = Math.round(100 * Number(v)) + "%";
      else if (b.kind === "slot") showSlot(b, v);
      else if (b.kind === "item") b.el.src = iconSrc(v, b.size);
      else if (b.kind === "on") {
        var has = b.el.className.indexOf(" on") >= 0;
        if (v && !has) b.el.className += " on";
        if (!v && has) b.el.className = b.el.className.replace(" on", "");
      }
    });
  }

  // Size classes for the CSS: the window width in rem, since Ore UI scales rem with the GUI scale.
  function measure() {
    var rem = 16;
    try {
      rem = parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
    } catch (e) {}
    var w = window.innerWidth / rem;
    var cls = w >= 100 ? "mc-w-l" : w >= 64 ? "mc-w-m" : "mc-w-s";
    ["mc-w-s", "mc-w-m", "mc-w-l"].forEach(function (c) {
      if (c !== cls) document.body.classList.remove(c);
    });
    document.body.classList.add(cls);
  }

  // fitLobbyHead keeps the game's name at the top middle of a lobby screen (ui.lobbySkin) clear of
  // the window: where the window, its tabs or its control bar would cover the logo and the name (what
  // oui-mc.css .mc-lobby-head holds), the window moves down when the screen has room below it, and the
  // name is hidden when it has not (a short screen). It runs once for each window drawn.
  function fitLobbyHead() {
    var head = root && root.querySelector(".mc-lobby-head");
    var win = root && root.querySelector(".mc-window");
    if (!head || !win || win.__lobbyFit) return;
    win.__lobbyFit = true;
    var left = Infinity, right = -Infinity, bottom = -Infinity;
    for (var i = 0; i < head.children.length; i++) {
      var h = head.children[i].getBoundingClientRect();
      left = Math.min(left, h.left);
      right = Math.max(right, h.right);
      bottom = Math.max(bottom, h.bottom);
    }
    if (bottom < 0) return;
    var top = Infinity;
    [win, root.querySelector(".mc-tabs"), root.querySelector(".mc-toolbar")].forEach(function (e) {
      if (!e) return;
      var r = e.getBoundingClientRect();
      if (r.width > 0 && r.left < right && r.right > left) top = Math.min(top, r.top);
    });
    var need = bottom - top;
    if (need <= 0) return;
    if (need <= window.innerHeight - win.getBoundingClientRect().bottom) {
      // the window is centred, so a margin moves it down by half of itself
      win.style.marginTop = Math.ceil(2 * need) + "px";
    } else {
      head.style.visibility = "hidden";
    }
  }

  function render() {
    measure();
    var active = state.screen === SCREEN && state.layout;
    if (active) document.body.classList.add("oui-active");
    else document.body.classList.remove("oui-active");
    if (!root) {
      root = el("div");
      root.id = "oui-ddui";
      document.body.appendChild(root);
    }
    hideTip();
    armed = null;
    rawDown = null;
    hoverSlot = null;
    cursors = [];
    root.innerHTML = "";
    bound = [];
    bars = [];
    inputs = [];
    pressables = [];
    if (kbFocus) kbFocus = null;
    regions = {};
    regionInk = {};
    regionRecs = {};
    tree = null;
    if (active) {
      tree = make(state.layout);
      root.appendChild(tree.el);
    }
    keepFocus();
    scrollToEnds();
    drawStatus();
  }

  // update shows a new layout of the form on the screen: patched into what is drawn, or drawn anew.
  function update() {
    if (!tree || !root || state.screen !== SCREEN) return render();
    measure();
    var r = patch(tree, state.layout, "dark");
    if (r !== tree) {
      root.replaceChild(r.el, tree.el);
      tree = r;
    }
    prune();
    scrollToEnds();
    drawStatus();
  }

  // scrollToEnds scrolls the boxes with `end` (oui-end) to their bottom, once the layout has its
  // sizes.
  function scrollToEnds() {
    setTimeout(function () {
      var ends = root ? root.querySelectorAll(".oui-end") : [];
      for (var i = 0; i < ends.length; i++) ends[i].scrollTop = ends[i].scrollHeight;
      bars.forEach(updateBar);
    }, 0);
  }

  // The debug line shows while the variable __debug is true.
  function drawStatus() {
    if (!status) {
      status = el("div");
      status.id = "oui-ddui-status";
      document.body.appendChild(status);
    }
    status.style.display = state.vars.__debug ? "flex" : "none";
    if (!state.vars.__debug) return;
    status.innerHTML = "";
    [
      "oui-ddui screen=" + (state.screen || "-") + " layout=" + state.layoutText.length + "B updates=" + state.updates +
        " vars=" + Object.keys(state.vars).length + " sent=" + state.seq + (state.error ? " error=" + state.error : ""),
      "localeFonts=" + cssVar("--localeSpecificNotoSansFonts") + " html=" + (document.documentElement.lang || "-") + " ." +
        (document.documentElement.className || "-") + " body=" + (document.body.className || "-"),
      "rootFont=" + fontOf(document.getElementById("root")) + " bodyFont=" + fontOf(document.body),
      "fontFaces=" + fontFaces(),
      "w=" + window.innerWidth + " h=" + window.innerHeight + " rem=" + remPx() + " ua=" + (navigator.userAgent || "-")
    ].forEach(function (line) {
      status.appendChild(el("div", null, line));
    });
  }

  // The values the debug line shows about fonts; "?" where the engine would not tell.
  function cssVar(name) {
    try {
      return window.getComputedStyle(document.documentElement).getPropertyValue(name).trim() || "-";
    } catch (e) {
      return "?";
    }
  }

  function fontOf(e) {
    try {
      return e ? window.getComputedStyle(e).fontFamily || "-" : "-";
    } catch (err) {
      return "?";
    }
  }

  function remPx() {
    try {
      return window.getComputedStyle(document.documentElement).fontSize;
    } catch (e) {
      return "?";
    }
  }

  // fontFaces are the families of the @font-face rules of the page's style sheets.
  function fontFaces() {
    var out = [];
    try {
      for (var i = 0; i < document.styleSheets.length; i++) {
        var rules = document.styleSheets[i].cssRules || [];
        for (var j = 0; j < rules.length; j++) {
          var r = rules[j];
          if (r.type !== 5) continue;
          var f = (r.style && (r.style.getPropertyValue("font-family") || r.style.fontFamily)) || "";
          f = f.replace(/["']/g, "");
          if (f && out.indexOf(f) < 0) out.push(f);
        }
      }
    } catch (e) {
      return "? " + e;
    }
    return out.length ? out.join(", ") : "-";
  }

  // --- keyboard --------------------------------------------------------------------------------

  // The keys a screen can answer: the arrow keys and WASD, Enter and the space bar.
  var KEY_NAMES = { 38: "up", 87: "up", 40: "down", 83: "down", 37: "left", 65: "left", 39: "right", 68: "right", 13: "ok", 32: "ok" };
  // A held key presses at most this often (Gameface may not mark the repeats of a held key).
  var KEY_REPEAT_MS = 120;
  var keyAt = {}; // name -> when the key last pressed

  // onKey presses what the screen's root gives the key in `keys` ({name: id}).
  function onKey(ev) {
    var name = KEY_NAMES[ev.keyCode];
    if (!name || state.screen !== SCREEN || !state.layout) return;
    var t = ev.target && ev.target.tagName;
    if (t === "INPUT" || t === "TEXTAREA") return;
    var now = Date.now();
    if (now - (keyAt[name] || 0) < KEY_REPEAT_MS) return;
    keyAt[name] = now;
    var keys = state.layout.keys;
    if (ev.preventDefault) ev.preventDefault();
    if (keys && keys[name]) send(keys[name], "click", null);
    else navKey(name);
  }

  // The keyboard selection: a key the screen does not answer moves a selection over the elements that
  // take presses, drawn like the pointer over them (oui-kb, with the tooltip), and Enter or the space
  // bar presses it. The pointer moving takes the selection away. When the selected element goes (the
  // press drew another screen), the element nearest to where it was is selected.
  var kbFocus = null; // the entry of pressables selected
  var kbOn = false; // whether the keyboard moved the selection since the pointer last moved
  var kbAt = null; // {x, y}: the middle of the element last selected

  function navigable(p) {
    if (!root || !root.contains(p.e)) return false;
    var n = p.nodeOf ? p.nodeOf() : null;
    if (n && (!n.id || n.press === false)) return false;
    var r = p.e.getBoundingClientRect();
    return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0 && r.top < window.innerHeight && r.left < window.innerWidth;
  }

  function middle(e) {
    var r = e.getBoundingClientRect();
    return { x: (r.left + r.right) / 2, y: (r.top + r.bottom) / 2, r: r };
  }

  function navKey(name) {
    if (name === "ok") {
      if (kbFocus && navigable(kbFocus)) kbFocus.fn({});
      return;
    }
    var list = pressables.filter(navigable);
    if (!list.length) return;
    kbOn = true;
    if (!kbFocus || !navigable(kbFocus)) return select(nearest(list, kbAt));
    var from = middle(kbFocus.e), best = null, bestScore = Infinity;
    list.forEach(function (p) {
      if (p === kbFocus) return;
      var to = middle(p.e);
      var along, across, overlap;
      if (name === "left" || name === "right") {
        along = name === "right" ? to.x - from.x : from.x - to.x;
        overlap = to.r.top < from.r.bottom && from.r.top < to.r.bottom;
        across = overlap ? 0 : Math.abs(to.y - from.y);
      } else {
        along = name === "down" ? to.y - from.y : from.y - to.y;
        overlap = to.r.left < from.r.right && from.r.left < to.r.right;
        across = overlap ? 0 : Math.abs(to.x - from.x);
      }
      if (along <= 1) return;
      var score = along + 2 * across;
      if (score < bestScore) {
        bestScore = score;
        best = p;
      }
    });
    if (best) select(best);
  }

  // nearest is the entry of list nearest to the point at, or the one at the top left without one.
  function nearest(list, at) {
    var best = null, bestScore = Infinity;
    list.forEach(function (p) {
      var m = middle(p.e);
      var score = at ? Math.abs(m.x - at.x) + Math.abs(m.y - at.y) : m.y * 4 + m.x;
      if (score < bestScore) {
        bestScore = score;
        best = p;
      }
    });
    return best;
  }

  function select(p) {
    if (kbFocus && kbFocus.e !== p.e) unmark(kbFocus.e);
    kbFocus = p;
    if (p.e.className.indexOf(" oui-kb") < 0) p.e.className += " oui-kb";
    revealIn(p.e);
    var m = middle(p.e);
    kbAt = { x: m.x, y: m.y };
    var n = p.nodeOf ? p.nodeOf() : null;
    if (n && n.tip) showTip(p.e, n.tip);
    else hideTip();
  }

  function unmark(e) {
    e.className = e.className.replace(/ oui-kb/g, "");
    if (tipOwner === e) hideTip();
  }

  // revealIn scrolls the scrolling boxes around e so that e shows.
  function revealIn(e) {
    for (var a = e.parentNode; a && a !== root; a = a.parentNode) {
      if (!a.className || !/(^| )(oui-scroll|mc-scroll)( |$)/.test(a.className) || a.scrollHeight <= a.clientHeight) continue;
      var r = e.getBoundingClientRect(), box = a.getBoundingClientRect();
      if (r.top < box.top) a.scrollTop -= box.top - r.top;
      else if (r.bottom > box.bottom) a.scrollTop += r.bottom - box.bottom;
    }
  }

  // keepFocus selects again, once the screen is laid out, after a patch or a new screen: the element
  // selected (a patch may have reset its class) or the one nearest to where it was.
  function keepFocus() {
    if (!kbOn) return;
    setTimeout(function () {
      if (!kbOn || !root) return;
      if (kbFocus && navigable(kbFocus)) {
        if (kbFocus.e.className.indexOf(" oui-kb") < 0) kbFocus.e.className += " oui-kb";
        return;
      }
      kbFocus = null;
      var list = pressables.filter(navigable);
      if (list.length) select(nearest(list, kbAt));
    }, 30);
  }

  // A mouse moving takes the keyboard selection away. A mousemove where the pointer already was (the
  // screen changed under it) does not.
  var mouseAt = null;
  function dropFocus(ev) {
    var at = ev && ev.clientX !== undefined ? ev.clientX + "," + ev.clientY : "";
    var moved = ev.type !== "mousemove" || (mouseAt !== null && at !== mouseAt);
    mouseAt = at;
    if (!kbOn || !moved) return;
    kbOn = false;
    if (kbFocus) unmark(kbFocus.e);
    kbFocus = null;
  }

  // --- data ------------------------------------------------------------------------------------

  function start() {
    drawStatus();
    measure();
    window.addEventListener("resize", function () {
      measure();
      var head = root && root.querySelector(".mc-lobby-head");
      var win = root && root.querySelector(".mc-window");
      if (head && win) {
        head.style.visibility = "";
        win.style.marginTop = "";
        win.__lobbyFit = false;
      }
    });
    function drop(ev) {
      dragging = null;
      panning = null;
      if (!ev.touches || ev.touches.length < 2) pinching = null;
    }
    ["mousemove", "touchmove"].forEach(function (type) {
      document.addEventListener(type, dragBar);
      document.addEventListener(type, dragCanvas);
    });
    document.addEventListener("touchstart", noteTouch);
    document.addEventListener("mousemove", dropFocus);
    document.addEventListener("touchstart", dropFocus);
    document.addEventListener("mousemove", noteMouse);
    ["mouseup", "touchend", "touchcancel"].forEach(function (type) {
      document.addEventListener(type, drop);
    });
    document.addEventListener("mouseup", releaseRaw);
    ["mousemove", "mousedown", "touchmove", "touchstart"].forEach(function (type) {
      document.addEventListener(type, trackPointer);
    });
    PROBE_TYPES.forEach(function (type) {
      document.addEventListener(type, probeEvent, true);
    });
    PROBE_WINDOW_TYPES.forEach(function (type) {
      window.addEventListener(type, probeEvent);
    });
    document.addEventListener("keydown", function (ev) {
      if (hoverSlot && ev.keyCode !== 27 && state.screen === SCREEN) send(hoverSlot, "slotkey", { k: ev.keyCode, shift: !!ev.shiftKey, ctrl: !!ev.ctrlKey });
    });
    // The vanilla inventory takes half a stack with the right button, which must not open a menu.
    document.addEventListener("contextmenu", function (ev) {
      if (state.screen === SCREEN && state.layout) ev.preventDefault();
    });
    // Content changes (regions, values, images loading) fire no scroll event, so the bars are also
    // checked on a timer. Bars of elements a region rebuild removed are dropped.
    setInterval(function () {
      bars = bars.filter(function (b) {
        return root && root.contains(b.sc);
      });
      bars.forEach(updateBar);
      fitLobbyHead();
      layoutPlanes();
      if (!envSent && probeLive()) drawProbe();
      if (probeLive()) probeGamepads();
    }, 100);
    // The vanilla form is hidden, so Esc would do nothing: report it and let the server close the form.
    document.addEventListener("keydown", function (ev) {
      if (ev.keyCode === 27 && state.screen === SCREEN && state.layout) send("close", "key", null);
    });
    document.addEventListener("keydown", onKey);
    if (!engine) {
      state.error = "no window.engine";
      drawStatus();
      return;
    }
    subscribe("vanilla.ddui.screenId", [], function (v) {
      v = unwrap(v);
      state.screen = v && typeof v === "object" ? v.screenId || "" : v || "";
      render();
    });
    subscribe("vanilla.ddui.string", [STORE, PROPERTY, "oui", false], function (v) {
      v = unwrap(v);
      state.updates++;
      if (typeof v !== "string" || v === "" || v === state.layoutText) return drawStatus();
      var again = false;
      try {
        var doc = JSON.parse(v);
        if (doc.form !== state.form) {
          state.form = doc.form;
          state.vars = {};
          applied = {};
        } else again = true;
        state.layout = doc.root;
        state.layoutText = v;
        keepSettings(doc.settings);
        state.error = "";
        held.forEach(take);
        held = [];
      } catch (e) {
        state.error = "layout: " + e;
      }
      // A layout of the form shown changes only what differs; another form's is drawn anew.
      if (again) update();
      else render();
    });
    function take(v) {
      v = unwrap(v);
      state.updates++;
      if (typeof v !== "string" || v === "") return drawStatus();
      var msg;
      try {
        msg = JSON.parse(v);
      } catch (e) {
        state.error = "vars: " + e;
        return drawStatus();
      }
      if (msg.f !== state.form) {
        // A patch of a form whose layout has not come yet waits for it; older ones are dropped.
        if (held.length < 32) held.push(v);
        return drawStatus();
      }
      if (applied[msg.s]) return drawStatus();
      applied[msg.s] = true;
      for (var k in msg.p) state.vars[k] = msg.p[k];
      applyVars(msg.p);
      drawStatus();
    }
    for (var i = 0; i < VARS_SLOTS; i++) subscribe("vanilla.ddui.string", [STORE, VARS_PROPERTY + i, "v", false], take);
  }

  // keepSettings holds the client settings the server asks for ([{id, kind, value}], the ids of the game's
  // settings screen) at their values: it watches each through the settings query of its kind and writes it
  // through vanilla_menus_update_settings when it differs, so a setting the player changes goes back.
  var SETTING_QUERIES = { bool: "settingsBooleanQuery", option: "settingsOptionQuery", number: "settingsNumberQuery", string: "settingsStringQuery" };
  var SETTING_WRITES = { bool: "updateBoolean", option: "updateOption", number: "updateNumber", string: "updateString" };
  var kept = {};

  function keepSettings(list) {
    (list || []).forEach(function (s) {
      if (!SETTING_QUERIES[s.kind]) return;
      var k = kept[s.id];
      if (k) {
        k.want = s.value;
        return fixSetting(s.id);
      }
      kept[s.id] = { kind: s.kind, want: s.value, has: false, value: undefined };
      subscribe("vanilla.menus." + SETTING_QUERIES[s.kind], [s.id], function (v) {
        v = unwrap(v);
        var k = kept[s.id];
        k.has = true;
        k.value = v && typeof v === "object" ? v.value : v;
        fixSetting(s.id);
      });
    });
  }

  function fixSetting(id) {
    var k = kept[id];
    if (!k.has || k.value === k.want) return;
    var write = command("vanilla_menus_update_settings", SETTING_WRITES[k.kind]);
    if (!write) return;
    try {
      write(id, k.want);
    } catch (e) {
      state.error = "setting " + id + ": " + e;
      drawStatus();
    }
  }

  if (document.body) start();
  else document.addEventListener("DOMContentLoaded", start);
})();
