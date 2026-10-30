// A stand-in for the game's Ore UI engine: answers the three queries the DDUI
// renderers subscribe to with the fixture in window.__FIXTURE__, and records
// what they write back.
(function () {
  var fx = window.__FIXTURE__ || { root: null, vars: {} };
  var FORM = "fx";
  var listeners = {};
  window.__written__ = [];
  window.__commands__ = {
    vanillaDduiDataStoreSetCommandGroup: {
      setDataStorePathString: {
        callable: function () {
          window.__written__.push(Array.prototype.slice.call(arguments));
        },
      },
    },
  };

  function answer(name, params) {
    if (name === "vanilla.ddui.screenId") return { screenId: "minecraft:custom_form" };
    if (name === "vanilla.ddui.string") {
      var prop = params[1];
      if (prop === "custom_form_data_1") return JSON.stringify({ form: FORM, root: fx.root });
      // The vars go in slot 0 with the fixture's form id only; the other slots stay empty.
      if (prop === "oui_vars_1_0") return JSON.stringify({ f: FORM, s: 1, p: fx.vars || {} });
    }
    return undefined;
  }

  window.engine = {
    on: function (ev, cb) {
      (listeners[ev] = listeners[ev] || []).push(cb);
    },
    trigger: function (ev) {
      var args = Array.prototype.slice.call(arguments, 1);
      var m = /^query:subscribe\/(.+)$/.exec(ev);
      if (!m) return;
      var id = args[0];
      var v = answer(m[1], args.slice(1));
      if (v === undefined) return;
      setTimeout(function () {
        (listeners["query:subscribed/" + id] || []).forEach(function (cb) {
          cb(v);
        });
      }, 0);
    },
  };
})();
