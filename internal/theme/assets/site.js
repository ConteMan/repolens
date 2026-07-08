(function () {
  "use strict";

  var statePrefix = "repolens:tree:";
  var scrollKey = "repolens:tree:scroll";
  // Cross-session preference for the wide-screen fixed sidebar:
  // "expanded" keeps the sidebar pinned, "collapsed" uses the overlay tree.
  var treePreferenceKey = "repolens:tree:preference";
  var floatingQuery = window.matchMedia ? window.matchMedia("(max-width: 1023px)") : null;
  var body = document.body;
  var treeButton = document.getElementById("btn-tree");
  var pinButton = document.getElementById("btn-pin-tree");
  var scrim = document.getElementById("scrim");
  var treeSource = document.getElementById("tree-src");
  var overlayTree = document.getElementById("overlay-tree");

  function storageGet(key) {
    try {
      return window.sessionStorage.getItem(key);
    } catch (_) {
      return null;
    }
  }

  function storageSet(key, value) {
    try {
      window.sessionStorage.setItem(key, value);
    } catch (_) {
      return;
    }
  }

  function localGet(key) {
    try {
      return window.localStorage.getItem(key);
    } catch (_) {
      return null;
    }
  }

  function localSet(key, value) {
    try {
      window.localStorage.setItem(key, value);
    } catch (_) {
      return;
    }
  }

  function keyFor(detail) {
    return statePrefix + (detail.getAttribute("data-tree-path") || "");
  }

  function applySavedDetailState(detail) {
    var saved = storageGet(keyFor(detail));
    if (saved === "open") {
      detail.open = true;
    } else if (saved === "closed") {
      detail.open = false;
    }
  }

  function syncDetailPath(path, open, origin) {
    if (!path) {
      return;
    }
    Array.prototype.forEach.call(document.querySelectorAll("details[data-tree-path]"), function (detail) {
      if (detail !== origin && detail.getAttribute("data-tree-path") === path) {
        detail.open = open;
      }
    });
  }

  function bindDetails(root) {
    Array.prototype.forEach.call(root.querySelectorAll("details[data-tree-path]"), function (detail) {
      applySavedDetailState(detail);
      detail.addEventListener("toggle", function () {
        var path = detail.getAttribute("data-tree-path") || "";
        storageSet(keyFor(detail), detail.open ? "open" : "closed");
        syncDetailPath(path, detail.open, detail);
      });
    });
  }

  function bindScroll(container) {
    if (!container) {
      return;
    }
    var savedScroll = parseInt(storageGet(scrollKey) || "0", 10);
    if (!Number.isNaN(savedScroll)) {
      container.scrollTop = savedScroll;
    }
    var pending = false;
    container.addEventListener("scroll", function () {
      if (pending) {
        return;
      }
      pending = true;
      window.requestAnimationFrame(function () {
        pending = false;
        storageSet(scrollKey, String(container.scrollTop));
      });
    }, { passive: true });
  }

  function isFloatingViewport() {
    return floatingQuery ? floatingQuery.matches : false;
  }

  function savedTreePreference() {
    return localGet(treePreferenceKey) === "collapsed" ? "collapsed" : "expanded";
  }

  function setOverlay(open) {
    var wasOpen = body.hasAttribute("data-overlay");
    if (open) {
      body.setAttribute("data-overlay", "open");
      // 焦点移入覆盖层首个可聚焦项，键盘用户可直接导航。
      var first = overlayTree && overlayTree.querySelector("a, button, summary");
      if (first && first.focus) {
        first.focus();
      }
    } else {
      body.removeAttribute("data-overlay");
      // 仅在真正关闭时归还焦点，避免初始化时抢走页面焦点。
      if (wasOpen && treeButton && treeButton.focus) {
        treeButton.focus();
      }
    }
  }

  function applyTreeMode(preference) {
    var floating = isFloatingViewport();
    body.setAttribute("data-tree", preference);
    if (floating) {
      body.setAttribute("data-tree-mode", "floating");
    } else {
      body.removeAttribute("data-tree-mode");
    }
    if (treeButton) {
      treeButton.setAttribute("aria-pressed", String(preference === "expanded" && !floating));
    }
  }

  function setTreePreference(preference) {
    localSet(treePreferenceKey, preference);
    applyTreeMode(preference);
  }

  function initTree() {
    if (!treeSource || !overlayTree) {
      bindDetails(document);
      bindScroll(document.querySelector("[data-tree-scroll]"));
      return;
    }

    overlayTree.innerHTML = treeSource.innerHTML;
    bindDetails(document);
    Array.prototype.forEach.call(document.querySelectorAll("[data-tree-scroll]"), bindScroll);
    applyTreeMode(savedTreePreference());

    if (treeButton) {
      treeButton.addEventListener("click", function () {
        if (!isFloatingViewport() && savedTreePreference() === "expanded") {
          setTreePreference("collapsed");
          return;
        }
        setOverlay(true);
      });
    }

    if (pinButton) {
      pinButton.addEventListener("click", function () {
        setOverlay(false);
        setTreePreference("expanded");
      });
    }

    if (scrim) {
      scrim.addEventListener("click", function () {
        setOverlay(false);
      });
    }

    overlayTree.addEventListener("click", function (event) {
      var target = event.target;
      if (target && target.closest && target.closest("a")) {
        setOverlay(false);
      }
    });

    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape") {
        setOverlay(false);
      }
    });

    if (floatingQuery) {
      var onViewportChange = function () {
        applyTreeMode(savedTreePreference());
        setOverlay(false);
      };
      if (typeof floatingQuery.addEventListener === "function") {
        floatingQuery.addEventListener("change", onViewportChange);
      } else if (typeof floatingQuery.addListener === "function") {
        floatingQuery.addListener(onViewportChange);
      }
    }
  }

  function initMermaid() {
    if (!window.mermaid || !document.querySelector(".mermaid")) {
      return;
    }
    window.mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "default"
    });
    if (typeof window.mermaid.run === "function") {
      window.mermaid.run({ querySelector: ".mermaid" });
    } else if (typeof window.mermaid.init === "function") {
      window.mermaid.init(undefined, document.querySelectorAll(".mermaid"));
    }
  }

  initTree();

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initMermaid);
  } else {
    initMermaid();
  }
}());
