(function () {
  "use strict";

  var body = document.body;
  var statePrefix = "repolens:tree:";
  var scrollKey = "repolens:tree:scroll";
  var treePreferenceKey = "repolens:tree:preference";
  var tocKey = "repolens:toc";
  var zoomKey = "repolens:zoom";
  var widthKey = "repolens:width";
  var zooms = [0.9, 1, 1.1, 1.25];
  var widths = ["narrow", "default", "full"];
  var floatingQuery = window.matchMedia ? window.matchMedia("(max-width: 1023px)") : null;
  var boundScrollContainers = typeof WeakSet === "function" ? new WeakSet() : null;
  var searchIndex = null, searchLoading = null, searchUnavailable = false, searchTrigger = null;
  var searchResults = [], searchSelected = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function storageGet(store, key) {
    try {
      return store.getItem(key);
    } catch (_) {
      return null;
    }
  }

  function storageSet(store, key, value) {
    try {
      store.setItem(key, value);
    } catch (_) {
      return;
    }
  }

  function sessionGet(key) {
    return storageGet(window.sessionStorage, key);
  }

  function sessionSet(key, value) {
    storageSet(window.sessionStorage, key, value);
  }

  function localGet(key) {
    return storageGet(window.localStorage, key);
  }

  function localSet(key, value) {
    storageSet(window.localStorage, key, value);
  }

  function keyFor(detail) {
    return statePrefix + (detail.getAttribute("data-tree-path") || "");
  }

  function applySavedDetailState(detail) {
    var saved = sessionGet(keyFor(detail));
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
        sessionSet(keyFor(detail), detail.open ? "open" : "closed");
        syncDetailPath(path, detail.open, detail);
      });
    });
  }

  function bindScroll(container) {
    if (!container || (boundScrollContainers && boundScrollContainers.has(container))) {
      return;
    }
    if (boundScrollContainers) {
      boundScrollContainers.add(container);
    }
    var savedScroll = parseInt(sessionGet(scrollKey) || "0", 10);
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
        sessionSet(scrollKey, String(container.scrollTop));
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
    var treeButton = $("btn-tree");
    var overlayTree = $("overlay-tree");
    var wasOpen = body.hasAttribute("data-overlay");
    if (open) {
      body.setAttribute("data-overlay", "open");
      var first = overlayTree && overlayTree.querySelector("a, button, summary");
      if (first && first.focus) {
        first.focus();
      }
    } else {
      body.removeAttribute("data-overlay");
      if (wasOpen && treeButton && treeButton.focus) {
        treeButton.focus();
      }
    }
  }

  function applyTreeMode(preference) {
    var floating = isFloatingViewport();
    var treeButton = $("btn-tree");
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

  function setupTreeDOM() {
    var treeSource = $("tree-src");
    var overlayTree = $("overlay-tree");
    if (treeSource && overlayTree) {
      overlayTree.innerHTML = treeSource.innerHTML;
      bindDetails(treeSource);
      bindDetails(overlayTree);
    } else {
      bindDetails(document);
    }
    Array.prototype.forEach.call(document.querySelectorAll("[data-tree-scroll]"), bindScroll);
    initTreeSearchEntrypoints(document);
    applyTreeMode(savedTreePreference());
  }

  function getZoomIndex() {
    var saved = parseInt(localGet(zoomKey) || "1", 10);
    if (Number.isNaN(saved)) {
      return 1;
    }
    return Math.max(0, Math.min(zooms.length - 1, saved));
  }

  function setZoomIndex(index) {
    localSet(zoomKey, String(index));
    applyZoom();
  }

  function applyZoom() {
    var index = getZoomIndex();
    var content = $("content");
    var readout = $("zoom-readout");
    if (content) {
      content.style.setProperty("--zoom", zooms[index]);
    }
    if (readout) {
      readout.textContent = Math.round(zooms[index] * 100) + "%";
    }
  }

  function widthLabels() {
    var zh = (document.documentElement.lang || "").toLowerCase().indexOf("zh") === 0;
    return zh ? ["窄栏", "默认", "全宽"] : ["Narrow", "Default", "Full"];
  }

  function applyWidth() {
    var saved = localGet(widthKey);
    var mode = widths.indexOf(saved) >= 0 ? saved : "default";
    var label = $("width-label");
    body.setAttribute("data-width", mode);
    if (label) {
      label.textContent = widthLabels()[widths.indexOf(mode)];
    }
  }

  function cycleWidth() {
    var current = body.getAttribute("data-width") || "default";
    var next = widths[(widths.indexOf(current) + 1) % widths.length] || "default";
    localSet(widthKey, next);
    applyWidth();
  }

  function applyTOC() {
    var panel = $("toc-panel");
    var button = $("btn-toc");
    if (!panel || !button) {
      body.removeAttribute("data-toc");
      return;
    }
    var open = localGet(tocKey) === "open";
    if (open) {
      body.setAttribute("data-toc", "open");
    } else {
      body.removeAttribute("data-toc");
    }
    button.setAttribute("aria-pressed", String(open));
    updateTOCActive();
  }

  function setTOC(open) {
    localSet(tocKey, open ? "open" : "closed");
    applyTOC();
  }

  function updateTOCActive() {
    var panel = $("toc-panel");
    if (!panel) {
      return;
    }
    var current = null;
    Array.prototype.forEach.call(document.querySelectorAll("#content h1[id], #content h2[id], #content h3[id], #content h4[id], #content h5[id], #content h6[id]"), function (heading) {
      if (heading.getBoundingClientRect().top < 90) {
        current = heading.id;
      }
    });
    Array.prototype.forEach.call(panel.querySelectorAll("a"), function (link) {
      link.classList.toggle("active", link.getAttribute("href") === "#" + current);
    });
  }

  function closeInfo() {
    var button = $("btn-info");
    body.removeAttribute("data-info");
    if (button) {
      button.setAttribute("aria-pressed", "false");
    }
  }

  function closeDownload() {
    var wrap = $("dl-wrap");
    if (wrap) {
      wrap.setAttribute("data-open", "false");
    }
    var dlButton = $("btn-dl");
    if (dlButton) {
      dlButton.setAttribute("aria-expanded", "false");
    }
  }

  function copyPath() {
    var node = document.querySelector("[data-page-path]");
    var text = node ? node.textContent : "";
    if (!text) {
      return;
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).catch(function () {
        fallbackCopy(text);
      });
    } else {
      fallbackCopy(text);
    }
  }

  function fallbackCopy(text) {
    var input = document.createElement("textarea");
    input.value = text;
    input.setAttribute("readonly", "");
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.appendChild(input);
    input.select();
    try {
      document.execCommand("copy");
    } catch (_) {
      return;
    } finally {
      document.body.removeChild(input);
    }
  }

  function searchModal() { return $("search-modal"); }
  function searchText(key) {
    var modal = searchModal();
    return modal ? modal.getAttribute("data-search-" + key) || "" : "";
  }
  function emptySearch(text) {
    var box = $("search-results");
    if (!box) { return; }
    box.textContent = "";
    var empty = document.createElement("div");
    empty.className = "search-empty"; empty.textContent = text; box.appendChild(empty);
  }
  function icon(name) {
    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    var use = document.createElementNS("http://www.w3.org/2000/svg", "use");
    svg.setAttribute("class", "icon"); svg.setAttribute("aria-hidden", "true");
    use.setAttribute("href", "#" + name); svg.appendChild(use);
    return svg;
  }
  function markText(parent, text, query) {
    var start = text.toLowerCase().indexOf(query.toLowerCase());
    if (!query || start < 0) { parent.appendChild(document.createTextNode(text)); return; }
    parent.appendChild(document.createTextNode(text.slice(0, start)));
    var mark = document.createElement("mark");
    mark.textContent = text.slice(start, start + query.length); parent.appendChild(mark);
    parent.appendChild(document.createTextNode(text.slice(start + query.length)));
  }
  function basename(repoPath) {
    var parts = String(repoPath || "").split("/");
    return parts[parts.length - 1] || repoPath;
  }
  function loadSearchIndex() {
    var modal = searchModal();
    if (!modal) { return Promise.reject(new Error("search disabled")); }
    if (searchIndex) { return Promise.resolve(searchIndex); }
    if (searchLoading) { return searchLoading; }
    searchLoading = window.fetch(modal.getAttribute("data-search-src"), { credentials: "same-origin" }).then(function (response) {
      if (!response.ok) { throw new Error("HTTP " + response.status); }
      return response.json();
    }).then(function (data) {
      searchIndex = data && Array.isArray(data.docs) ? data : { docs: [] };
      searchIndex.docs.sort(function (a, b) { return String(a.path).localeCompare(String(b.path)); });
      // 上次失败后的重试成功要解除"不可用"标记，否则搜索永久失效。
      searchUnavailable = false;
      return searchIndex;
    }).catch(function (error) {
      searchUnavailable = true;
      searchLoading = null;
      throw error;
    });
    return searchLoading;
  }
  function buildSearchResults(query) {
    var lower = query.toLowerCase(), files = [], headings = [];
    (searchIndex ? searchIndex.docs : []).forEach(function (doc) {
      var base = basename(doc.path);
      var label = doc.title && doc.title !== base ? base + " · " + doc.title : base;
      if ((label + " " + doc.path).toLowerCase().indexOf(lower) >= 0) { files.push({ type: "file", label: label, hint: doc.path, view: doc.view }); }
      (doc.headings || []).forEach(function (heading) {
        if (String(heading.text || "").toLowerCase().indexOf(lower) >= 0) { headings.push({ type: "heading", label: heading.text, hint: doc.path, view: doc.view, anchor: heading.anchor }); }
      });
    });
    return files.concat(headings);
  }
  function renderSearchResults(query) {
    var box = $("search-results");
    query = query.trim();
    if (!box) { return; }
    if (searchUnavailable || !query) {
      searchResults = []; searchSelected = 0;
      emptySearch(searchText(searchUnavailable ? "unavailable" : "intro"));
      return;
    }
    searchResults = buildSearchResults(query);
    searchSelected = 0; box.textContent = "";
    if (!searchResults.length) { emptySearch(searchText("empty")); return; }
    var lastType = "";
    searchResults.forEach(function (item, index) {
      if (item.type !== lastType) {
        var group = document.createElement("div");
        group.className = "search-group"; group.textContent = item.type === "file" ? searchText("files") : searchText("headings"); box.appendChild(group);
        lastType = item.type;
      }
      var row = document.createElement("div");
      var label = document.createElement("span"), hint = document.createElement("small");
      row.className = "search-item" + (index === searchSelected ? " selected" : ""); row.setAttribute("data-search-index", String(index));
      row.appendChild(icon(item.type === "file" ? "icon-file" : "icon-toc"));
      markText(label, item.label, query); hint.textContent = item.hint;
      row.appendChild(label); row.appendChild(hint); box.appendChild(row);
    });
  }
  function selectSearch(delta) {
    if (!searchResults.length) { return; }
    searchSelected = (searchSelected + delta + searchResults.length) % searchResults.length;
    Array.prototype.forEach.call(document.querySelectorAll(".search-item"), function (item, index) {
      item.classList.toggle("selected", index === searchSelected);
      if (index === searchSelected) { item.scrollIntoView({ block: "nearest" }); }
    });
  }
  function closeSearch() {
    var input = $("search-input");
    body.removeAttribute("data-search");
    searchResults = []; searchSelected = 0;
    if (input) { input.value = ""; }
    emptySearch(searchText("intro"));
    if (searchTrigger && document.body.contains(searchTrigger) && searchTrigger.focus) { searchTrigger.focus(); }
    searchTrigger = null;
  }
  function openSearch(trigger) {
    var input = $("search-input");
    if (!input) { return; }
    searchTrigger = trigger || document.activeElement;
    body.setAttribute("data-search", "open");
    setOverlay(false); closeInfo(); closeDownload();
    input.disabled = false;
    input.placeholder = searchText("placeholder");
    input.focus();
    loadSearchIndex().then(function () { renderSearchResults(input.value); }).catch(function () {
      input.value = ""; input.placeholder = searchText("unavailable"); input.disabled = true;
      emptySearch(searchText("unavailable"));
    });
  }
  function goSearchResult(item) {
    if (!item) { return; }
    var modal = searchModal();
    var target = (modal ? modal.getAttribute("data-search-root") || "" : "") + item.view + (item.anchor ? "#" + item.anchor : "");
    var url = sameOriginURL(target);
    closeSearch();
    if (!url) {
      window.location.href = target;
      return;
    }
    fetchPage(url, true).catch(function () { window.location.href = url.href; });
  }
  function initTreeSearchEntrypoints(root) {
    Array.prototype.forEach.call(root.querySelectorAll("[data-tree-search-placeholder]"), function (entry) {
      var label = document.createElement("span"), key = document.createElement("kbd");
      entry.hidden = false; entry.setAttribute("role", "button"); entry.setAttribute("tabindex", "0");
      entry.setAttribute("aria-label", searchText("label"));
      entry.textContent = "";
      label.textContent = searchText("placeholder"); key.textContent = "/";
      entry.appendChild(icon("icon-search")); entry.appendChild(label); entry.appendChild(key);
    });
  }

  function applyToolbarState() {
    applyZoom();
    applyWidth();
    applyTOC();
    closeInfo();
    closeDownload();
  }

  function sameOriginURL(href) {
    try {
      return new URL(href, window.location.href);
    } catch (_) {
      return null;
    }
  }

  function isPjaxLink(link) {
    var raw = link ? link.getAttribute("href") : "";
    if (!link || link.target || link.hasAttribute("download") || !raw || raw.charAt(0) === "#") {
      return false;
    }
    var url = sameOriginURL(raw);
    return !!url && url.origin === window.location.origin && url.pathname.indexOf("/view/") >= 0;
  }

  function fetchPage(url, push) {
    return window.fetch(url.href, { credentials: "same-origin" }).then(function (response) {
      if (!response.ok) {
        throw new Error("HTTP " + response.status);
      }
      return response.text();
    }).then(function (html) {
      var doc = new DOMParser().parseFromString(html, "text/html");
      var nextTopbar = doc.querySelector(".topbar");
      var nextContent = doc.querySelector("#content");
      var nextTree = doc.querySelector("#tree-src");
      if (!nextTopbar || !nextContent || !nextTree) {
        throw new Error("missing pjax targets");
      }
      document.title = doc.title;
      var nextKind = doc.body && doc.body.getAttribute("data-page-kind");
      if (nextKind) {
        body.setAttribute("data-page-kind", nextKind);
      }
      document.querySelector(".topbar").outerHTML = nextTopbar.outerHTML;
      $("content").outerHTML = nextContent.outerHTML;
      $("tree-src").innerHTML = nextTree.innerHTML;
      replaceOptional("#toc-panel", doc);
      replaceOptional("#search-modal", doc);
      loadMermaidIfNeeded(doc);
      setupTreeDOM();
      applyToolbarState();
      initMermaid();
      if (push) {
        window.history.pushState({ pjax: true }, "", url.href);
      }
      scrollAfterPjax(url);
    });
  }

  // pjax 换页后手动定位：带 fragment 滚到锚点（含搜索章节跳转），否则回页顶。
  function scrollAfterPjax(url) {
    var hash = url.hash ? decodeURIComponent(url.hash.slice(1)) : "";
    var target = hash ? document.getElementById(hash) : null;
    if (target && target.scrollIntoView) {
      target.scrollIntoView();
    } else {
      window.scrollTo(0, 0);
    }
  }

  function replaceOptional(selector, doc) {
    var current = document.querySelector(selector);
    var next = doc.querySelector(selector);
    if (current && next) {
      current.outerHTML = next.outerHTML;
    } else if (current) {
      current.remove();
    } else if (next) {
      document.querySelector(".overlay").insertAdjacentHTML("afterend", next.outerHTML);
    }
  }

  function loadMermaidIfNeeded(doc) {
    if (window.mermaid || !doc.querySelector('script[src$="mermaid.min.js"]')) {
      return;
    }
    var selfScript = document.querySelector('script[src$="site.js"]');
    if (!selfScript) {
      return;
    }
    var script = document.createElement("script");
    script.defer = true;
    script.src = selfScript.src.replace(/site\.js$/, "mermaid.min.js");
    script.onload = initMermaid;
    document.body.appendChild(script);
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

  document.addEventListener("click", function (event) {
    var target = event.target;
    var link = target.closest && target.closest("a");
    if (link && isPjaxLink(link) && !event.defaultPrevented && event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) {
      event.preventDefault();
      setOverlay(false);
      closeSearch();
      fetchPage(sameOriginURL(link.href), true).catch(function () {
        window.location.href = link.href;
      });
      return;
    }

    var button = target.closest && target.closest("button");
    if (button && button.id === "btn-tree") {
      if (!isFloatingViewport() && savedTreePreference() === "expanded") {
        setTreePreference("collapsed");
      } else {
        setOverlay(true);
      }
    } else if (button && button.id === "btn-pin-tree") {
      setOverlay(false);
      setTreePreference("expanded");
    } else if (button && button.id === "btn-back") {
      window.history.back();
    } else if (button && button.id === "btn-fwd") {
      window.history.forward();
    } else if (button && button.id === "btn-toc") {
      setTOC(body.getAttribute("data-toc") !== "open");
    } else if (button && button.id === "btn-zoom-out") {
      setZoomIndex(Math.max(0, getZoomIndex() - 1));
    } else if (button && button.id === "btn-zoom-in") {
      setZoomIndex(Math.min(zooms.length - 1, getZoomIndex() + 1));
    } else if (button && button.id === "btn-width") {
      cycleWidth();
    } else if (button && button.id === "btn-info") {
      if (body.getAttribute("data-info") === "open") {
        closeInfo();
      } else {
        body.setAttribute("data-info", "open");
        button.setAttribute("aria-pressed", "true");
        var infoPanel = document.querySelector(".info-panel");
        var infoFirst = infoPanel && infoPanel.querySelector("a, button");
        if (infoFirst && infoFirst.focus) {
          infoFirst.focus();
        }
      }
    } else if (button && button.id === "btn-dl") {
      var wrap = $("dl-wrap");
      if (wrap) {
        var dlOpen = wrap.getAttribute("data-open") === "true";
        wrap.setAttribute("data-open", dlOpen ? "false" : "true");
        button.setAttribute("aria-expanded", String(!dlOpen));
      }
    } else if (button && button.id === "btn-search") {
      openSearch(button);
      return;
    } else if (target.closest && target.closest("[data-tree-search-placeholder]")) {
      openSearch(target.closest("[data-tree-search-placeholder]"));
      return;
    } else if (target.closest && target.closest(".search-item")) {
      goSearchResult(searchResults[Number(target.closest(".search-item").getAttribute("data-search-index"))]);
      return;
    } else if (link && link.id === "info-copy") {
      event.preventDefault();
      copyPath();
    } else if (!target.closest("#info-wrap")) {
      closeInfo();
    }

    if (!target.closest("#dl-wrap")) {
      closeDownload();
    }
    if (target === $("scrim")) {
      setOverlay(false);
      closeSearch();
    }
  });

  document.addEventListener("input", function (event) {
    if (event.target && event.target.id === "search-input") {
      renderSearchResults(event.target.value);
    }
  });

  document.addEventListener("keydown", function (event) {
    var tag = event.target && event.target.tagName;
    var editable = /^(INPUT|TEXTAREA|SELECT)$/.test(tag) || event.target.isContentEditable;
    if (event.key === "/" && body.getAttribute("data-search") !== "open" && !editable) {
      event.preventDefault();
      openSearch(document.activeElement);
    } else if ((event.key === "Enter" || event.key === " ") && event.target.closest && event.target.closest("[data-tree-search-placeholder]")) {
      // 树顶搜索入口是 role="button" 的 div，键盘激活需自行处理。
      event.preventDefault();
      openSearch(event.target.closest("[data-tree-search-placeholder]"));
    } else if (body.getAttribute("data-search") === "open" && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
      event.preventDefault();
      selectSearch(event.key === "ArrowDown" ? 1 : -1);
    } else if (body.getAttribute("data-search") === "open" && event.key === "Enter") {
      event.preventDefault();
      goSearchResult(searchResults[searchSelected]);
    } else if (body.getAttribute("data-search") === "open" && event.key === "Escape") {
      event.preventDefault();
      closeSearch();
    } else if (event.key === "Escape") {
      var infoWasOpen = body.getAttribute("data-info") === "open";
      setOverlay(false);
      setTOC(false);
      closeInfo();
      closeDownload();
      // Esc 关闭 ⓘ 面板时把焦点还给触发按钮（外点关闭不抢焦点）。
      var infoButton = $("btn-info");
      if (infoWasOpen && infoButton && infoButton.focus) {
        infoButton.focus();
      }
    }
  });

  window.addEventListener("scroll", updateTOCActive, { passive: true });
  window.addEventListener("popstate", function () {
    fetchPage(new URL(window.location.href), false).catch(function () {
      window.location.reload();
    });
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

  setupTreeDOM();
  applyToolbarState();
  if (!window.history.state) {
    window.history.replaceState({ pjax: true }, "", window.location.href);
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initMermaid);
  } else {
    initMermaid();
  }
}());
