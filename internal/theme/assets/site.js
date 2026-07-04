(function () {
  "use strict";

  var statePrefix = "repolens:tree:";
  var scrollKey = "repolens:tree:scroll";
  var sidebar = document.querySelector("[data-tree-scroll]");
  var details = Array.prototype.slice.call(document.querySelectorAll("details[data-tree-path]"));

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

  function keyFor(detail) {
    return statePrefix + (detail.getAttribute("data-tree-path") || "");
  }

  details.forEach(function (detail) {
    var saved = storageGet(keyFor(detail));
    if (saved === "open") {
      detail.open = true;
    } else if (saved === "closed") {
      detail.open = false;
    }
    detail.addEventListener("toggle", function () {
      storageSet(keyFor(detail), detail.open ? "open" : "closed");
    });
  });

  if (sidebar) {
    var savedScroll = parseInt(storageGet(scrollKey) || "0", 10);
    if (!Number.isNaN(savedScroll)) {
      sidebar.scrollTop = savedScroll;
    }
    var pending = false;
    sidebar.addEventListener("scroll", function () {
      if (pending) {
        return;
      }
      pending = true;
      window.requestAnimationFrame(function () {
        pending = false;
        storageSet(scrollKey, String(sidebar.scrollTop));
      });
    }, { passive: true });
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

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initMermaid);
  } else {
    initMermaid();
  }
}());
