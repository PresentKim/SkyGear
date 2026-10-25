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
// More nodes for the Minecraft style (oui-mc.css): frame (an empty slot), arrow, img (icon, size).
// `ink: "panel"` on a node makes the text under it the dark variant of its § colours (text drawn right on
// the grey window, like common ui.OnPanel); buttons, slot counts and tooltips keep the colours.
// The body gets mc-w-s / mc-w-m / mc-w-l for the window width, so the CSS can lay out by size.
// `bind` names a variable that replaces the text, or the value of a bar. Text may carry § colour codes.
// An icon is "geo:<item name>" (drawn by the engine through geo-item://, the URL the vanilla container
// slots use), "tex:<texture path>" (a pack texture through pack://), or a bare item name.
// A region's children come from the variable "@<name>" (a node), so the server can replace one part of
// the screen without resending the layout; the region is patched like the layout. `tip` shows a tooltip while the pointer is over the node.
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
    s = s === undefined || s === null ? "" : String(s);
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

  // showTip shows text as the tooltip of e, right of it.
  function showTip(e, text) {
    if (!tip) {
      tip = el("div");
      tip.id = "oui-tip";
      document.body.appendChild(tip);
    }
    tipOwner = e;
    rich(tip, text, false);
    var r = e.getBoundingClientRect();
    tip.style.left = Math.round(r.right + 6) + "px";
    tip.style.top = Math.round(r.top) + "px";
    tip.style.display = "flex";
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
      case "canvas":
        e = buildCanvas(n, rec);
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

  // canvas is a view on a plane of absolutely placed children that the player drags around. Where it
  // was dragged to is kept by its key, so a screen drawn again (a quest opened) stays where it was;
  // the first time the point (focusX, focusY) in rem is put in the middle.
  var pans = {};

  function buildCanvas(n, rec) {
    var view = el("div", "oui-canvas");
    var plane = el("div", "oui-canvas-plane");
    view.appendChild(plane);
    appendChildren(plane, n.children, rec);
    var pan = pans[n.key];
    function place() {
      plane.style.left = pan.x + "px";
      plane.style.top = pan.y + "px";
    }
    setTimeout(function () {
      if (!pan) {
        var rem = parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
        pan = pans[n.key] = { x: view.clientWidth / 2 - (n.focusX || 0) * rem, y: view.clientHeight / 2 - (n.focusY || 0) * rem };
      }
      place();
    }, 0);
    onDragStart(view, function (ev) {
      if (ev.target !== view && ev.target !== plane && !/oui-line-seg|oui-canvas-deco/.test(ev.target.className)) return;
      if (!pan) return;
      if (tip) tip.style.display = "none";
      var at = pointOf(ev);
      panning = { pan: pan, place: place, x: at.clientX, y: at.clientY, px: pan.x, py: pan.y };
    });
    return view;
  }

  var panning = null; // {pan, place, x, y, px, py}: the canvas being dragged

  function dragCanvas(ev) {
    if (!panning) return;
    var at = pointOf(ev);
    panning.pan.x = panning.px + at.clientX - panning.x;
    panning.pan.y = panning.py + at.clientY - panning.y;
    panning.place();
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
    box: { "class": true, style: true, on: true },
    text: { "class": true, style: true, text: true },
    img: { "class": true, style: true, icon: true, size: true },
    canvas: { focusX: true, focusY: true } // only the first drawing reads the focus
  };

  // Node types whose presses read the id from the record (the others keep the id they were built with).
  var LIVE_ID = { box: true, button: true, col: true, row: true, grid: true, text: true, img: true, region: true, frame: true, arrow: true, space: true, canvas: true };

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
    if (tip && tipOwner && !root.contains(tipOwner)) hideTip();
    if (armed && !root.contains(armed)) armed = null;
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
    onPress(box, function () {
      send(n.id, "click", null);
    });
    return box;
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
    if (n.placeholder) input.placeholder = n.placeholder;
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
      var b = el("div", "oui-button" + (opt === n.value ? " on" : ""), opt);
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
    root.innerHTML = "";
    bound = [];
    bars = [];
    inputs = [];
    regions = {};
    regionInk = {};
    regionRecs = {};
    tree = null;
    if (active) {
      tree = make(state.layout);
      root.appendChild(tree.el);
    }
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
    drawStatus();
  }

  // The debug line shows while the variable __debug is true.
  function drawStatus() {
    if (!status) {
      status = el("div");
      status.id = "oui-ddui-status";
      document.body.appendChild(status);
    }
    status.style.display = state.vars.__debug ? "flex" : "none";
    status.textContent =
      "oui-ddui screen=" + (state.screen || "-") + " layout=" + state.layoutText.length + "B updates=" + state.updates +
      " vars=" + Object.keys(state.vars).length + " sent=" + state.seq + (state.error ? " error=" + state.error : "");
  }

  // --- data ------------------------------------------------------------------------------------

  function start() {
    drawStatus();
    measure();
    window.addEventListener("resize", measure);
    function drop() {
      dragging = null;
      panning = null;
    }
    ["mousemove", "touchmove"].forEach(function (type) {
      document.addEventListener(type, dragBar);
      document.addEventListener(type, dragCanvas);
    });
    document.addEventListener("touchstart", noteTouch);
    document.addEventListener("mousemove", noteMouse);
    ["mouseup", "touchend", "touchcancel"].forEach(function (type) {
      document.addEventListener(type, drop);
    });
    // Content changes (regions, values, images loading) fire no scroll event, so the bars are also
    // checked on a timer. Bars of elements a region rebuild removed are dropped.
    setInterval(function () {
      bars = bars.filter(function (b) {
        return root && root.contains(b.sc);
      });
      bars.forEach(updateBar);
    }, 100);
    // The vanilla form is hidden, so Esc would do nothing: report it and let the server close the form.
    document.addEventListener("keydown", function (ev) {
      if (ev.keyCode === 27 && state.screen === SCREEN && state.layout) send("close", "key", null);
    });
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

  if (document.body) start();
  else document.addEventListener("DOMContentLoaded", start);
})();
