// Fog-of-War Canvas + Token Layer
(function () {
  "use strict";

  // --- Zoom/Pan Controller ---
  function createZoomController(canvasWrap) {
    var zoomWrap = canvasWrap.querySelector(".zoom-wrap");
    if (!zoomWrap) return;

    var scale = 1;
    var panX = 0;
    var panY = 0;
    var minScale = 1;
    var maxScale = 5;
    var panning = false;
    var panStartX = 0;
    var panStartY = 0;
    var lastPanX = 0;
    var lastPanY = 0;

    function applyTransform() {
      zoomWrap.style.transform = "scale(" + scale + ") translate(" + panX + "px, " + panY + "px)";
    }

    function clamp(val, lo, hi) {
      return Math.max(lo, Math.min(hi, val));
    }

    // Wheel to zoom toward cursor
    canvasWrap.addEventListener("wheel", function (e) {
      e.preventDefault();
      var oldScale = scale;
      var factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
      scale = clamp(scale * factor, minScale, maxScale);

      var wrapRect = canvasWrap.getBoundingClientRect();
      var cx = e.clientX - wrapRect.left;
      var cy = e.clientY - wrapRect.top;

      panX = cx / scale - cx / oldScale + panX;
      panY = cy / scale - cy / oldScale + panY;

      applyTransform();
    }, { passive: false });

    // Middle-mouse drag to pan
    canvasWrap.addEventListener("mousedown", function (e) {
      if (e.button === 1) {
        e.preventDefault();
        panning = true;
        panStartX = e.clientX;
        panStartY = e.clientY;
        lastPanX = panX;
        lastPanY = panY;
      }
    });

    window.addEventListener("mousemove", function (e) {
      if (!panning) return;
      panX = lastPanX + (e.clientX - panStartX) / scale;
      panY = lastPanY + (e.clientY - panStartY) / scale;
      applyTransform();
    });

    window.addEventListener("mouseup", function (e) {
      if (e.button === 1) panning = false;
    });

    // Double-click to reset
    canvasWrap.addEventListener("dblclick", function () {
      scale = 1;
      panX = 0;
      panY = 0;
      applyTransform();
    });

    // Prevent context menu on middle click
    canvasWrap.addEventListener("auxclick", function (e) {
      if (e.button === 1) e.preventDefault();
    });
  }

  // --- Token Layer (shared between DM and Player) ---
  function createTokenLayer(container, mapImg, mapId, isDM) {
    var canvas = document.createElement("canvas");
    canvas.className = "token-canvas";
    container.querySelector(".zoom-wrap").appendChild(canvas);
    var ctx = canvas.getContext("2d");
    var tokens = [];
    var dragging = null;
    var dragOffset = { x: 0, y: 0 };
    var hasDragged = false;
    var dragStartPos = { x: 0, y: 0 };
    var activePopup = null;

    // Bug 2: alive flag — set false when container is removed by HTMX
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    // In fog mode, token canvas doesn't receive pointer events (DM only)
    if (isDM) canvas.style.pointerEvents = "none";

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      render();
    }

    function coords(e) {
      var rect = canvas.getBoundingClientRect();
      return {
        x: (e.clientX - rect.left) * (canvas.width / rect.width),
        y: (e.clientY - rect.top) * (canvas.height / rect.height),
      };
    }

    function render() {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      tokens.forEach(function (t) {
        if (!isDM && !t.visible) return;
        var alpha = isDM && !t.visible ? 0.4 : 0.8;
        ctx.beginPath();
        ctx.arc(t.x, t.y, t.radius, 0, Math.PI * 2);
        ctx.fillStyle = t.color || "#e94560";
        ctx.globalAlpha = alpha;
        ctx.fill();
        ctx.strokeStyle = "#fff";
        ctx.lineWidth = 2;
        ctx.stroke();
        if (t.label) {
          ctx.fillStyle = "#fff";
          ctx.font = Math.max(12, t.radius * 0.6) + "px sans-serif";
          ctx.textAlign = "center";
          ctx.textBaseline = "middle";
          ctx.fillText(t.label, t.x, t.y);
        }
        ctx.globalAlpha = 1;
      });
    }

    function loadTokens() {
      var url = isDM
        ? "/dm/maps/" + mapId + "/tokens"
        : "/maps/" + mapId + "/tokens";
      fetch(url)
        .then(function (r) { return r.json(); })
        .then(function (data) {
          tokens = data || [];
          render();
        });
    }

    function hitTest(pos) {
      for (var i = tokens.length - 1; i >= 0; i--) {
        var t = tokens[i];
        if (!isDM && !t.visible) continue;
        var dx = pos.x - t.x;
        var dy = pos.y - t.y;
        if (dx * dx + dy * dy <= t.radius * t.radius) {
          return t;
        }
      }
      return null;
    }

    function setMode(mode) {
      canvas.style.pointerEvents = mode === "tokens" ? "auto" : "none";
    }

    // --- Mouse handlers (work for both DM token mode and player token dragging) ---
    canvas.addEventListener("mousedown", function (e) {
      if (e.button !== 0) return;
      var pos = coords(e);
      var token = hitTest(pos);
      if (token && (isDM || token.moveable)) {
        dragging = token;
        hasDragged = false;
        dragStartPos.x = pos.x;
        dragStartPos.y = pos.y;
        dragOffset.x = token.x - pos.x;
        dragOffset.y = token.y - pos.y;
      } else if (isDM && e.button === 0) {
        // Click on empty space — close popup, place a new token
        closeTokenPopup();
        addToken(pos);
      }
    });

    canvas.addEventListener("mousemove", function (e) {
      if (!dragging) return;
      var pos = coords(e);
      if (!hasDragged) {
        var dx = pos.x - dragStartPos.x;
        var dy = pos.y - dragStartPos.y;
        if (dx * dx + dy * dy < 9) return; // 3px threshold
        hasDragged = true;
      }
      dragging.x = pos.x + dragOffset.x;
      dragging.y = pos.y + dragOffset.y;
      render();
    });

    canvas.addEventListener("mouseup", function (e) {
      if (!dragging) return;
      if (hasDragged) {
        // Drag completed — send position update
        var url = isDM
          ? "/dm/maps/" + mapId + "/tokens/" + dragging.id
          : "/maps/" + mapId + "/tokens/" + dragging.id;
        var body = new URLSearchParams();
        body.set("x", dragging.x);
        body.set("y", dragging.y);
        fetch(url, { method: "PUT", body: body });
      } else if (isDM) {
        // Click without drag — open edit popup
        showTokenPopup(dragging, e);
      }
      dragging = null;
    });

    // Right-click to delete token (DM only)
    if (isDM) {
      canvas.addEventListener("contextmenu", function (e) {
        var pos = coords(e);
        var token = hitTest(pos);
        if (token) {
          e.preventDefault();
          fetch("/dm/maps/" + mapId + "/tokens/" + token.id, {
            method: "DELETE",
          }).then(function () { loadTokens(); });
        }
      });
    }

    function getToolbarValue(attr, fallback) {
      var el = container.querySelector("[" + attr + "]");
      if (!el) return fallback;
      if (el.type === "checkbox") return el.checked;
      return el.value || fallback;
    }

    function addToken(pos) {
      var body = new URLSearchParams();
      body.set("x", pos.x);
      body.set("y", pos.y);
      body.set("radius", getToolbarValue("data-token-radius", "20"));
      body.set("label", getToolbarValue("data-token-label", ""));
      body.set("color", getToolbarValue("data-token-color", "#e94560"));
      body.set("visible", getToolbarValue("data-token-visible", false) ? "true" : "false");
      body.set("moveable", getToolbarValue("data-token-moveable", true) ? "true" : "false");
      fetch("/dm/maps/" + mapId + "/tokens", {
        method: "POST",
        body: body,
      }).then(function () { loadTokens(); });
    }

    function closeTokenPopup() {
      if (activePopup) {
        activePopup.remove();
        activePopup = null;
      }
    }

    function showTokenPopup(token, e) {
      closeTokenPopup();
      var rect = container.getBoundingClientRect();
      var popup = document.createElement("div");
      popup.className = "token-popup";
      popup.style.left = (e.clientX - rect.left + 10) + "px";
      popup.style.top = (e.clientY - rect.top + 10) + "px";
      popup.innerHTML =
        '<label>Name <input type="text" data-field="label" value="' + (token.label || "").replace(/"/g, "&quot;") + '" class="token-input-text"/></label>' +
        '<label>Color <input type="color" data-field="color" value="' + (token.color || "#e94560") + '"/></label>' +
        '<label>Size <input type="range" data-field="radius" min="5" max="100" value="' + (token.radius || 20) + '"/></label>' +
        '<label><input type="checkbox" data-field="visible"' + (token.visible ? " checked" : "") + '/> Visible</label>' +
        '<label><input type="checkbox" data-field="moveable"' + (token.moveable ? " checked" : "") + '/> Moveable</label>' +
        '<div class="token-popup-actions">' +
          '<button class="btn-sm" data-popup-save>Save</button>' +
          '<button class="btn-sm btn-danger" data-popup-delete>Delete</button>' +
        '</div>';

      popup.addEventListener("mousedown", function (ev) { ev.stopPropagation(); });

      popup.querySelector("[data-popup-save]").addEventListener("click", function () {
        var body = new URLSearchParams();
        body.set("label", popup.querySelector('[data-field="label"]').value);
        body.set("color", popup.querySelector('[data-field="color"]').value);
        body.set("radius", popup.querySelector('[data-field="radius"]').value);
        body.set("visible", popup.querySelector('[data-field="visible"]').checked ? "true" : "false");
        body.set("moveable", popup.querySelector('[data-field="moveable"]').checked ? "true" : "false");
        fetch("/dm/maps/" + mapId + "/tokens/" + token.id, {
          method: "PUT",
          body: body,
        }).then(function () {
          closeTokenPopup();
          loadTokens();
        });
      });

      popup.querySelector("[data-popup-delete]").addEventListener("click", function () {
        fetch("/dm/maps/" + mapId + "/tokens/" + token.id, {
          method: "DELETE",
        }).then(function () {
          closeTokenPopup();
          loadTokens();
        });
      });

      container.appendChild(popup);
      activePopup = popup;
    }

    // Bug 2: Guard SSE and resize listeners with alive check
    document.body.addEventListener("sse:tokenUpdate", function () {
      if (!alive) return;
      loadTokens();
    });

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);
    window.addEventListener("resize", function () {
      if (!alive) return;
      resize();
    });
    loadTokens();

    return { refresh: loadTokens, resize: resize, setMode: setMode };
  }

  // --- DM Fog Canvas ---
  function initDMCanvas(container, mapId) {
    var mapImg = container.querySelector(".map-image");
    if (!mapImg) return;

    // Bug 2: Remove any existing fog/token canvases left over from a previous init
    var zoomWrap = container.querySelector(".zoom-wrap");
    var oldFog = zoomWrap.querySelector(".fog-canvas");
    if (oldFog) oldFog.remove();
    var oldToken = zoomWrap.querySelector(".token-canvas");
    if (oldToken) oldToken.remove();

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    zoomWrap.appendChild(canvas);
    var ctx = canvas.getContext("2d");

    var tool = "reveal";
    var shape = "brush";
    var brushSize = 40;
    var drawing = false;
    var rectStart = null;
    var fogSnapshot = null;
    // Bug 1: track last brush position for interpolation
    var lastBrushPos = null;

    // Bug 2: alive flag — set false when container is cleaned up by HTMX
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      loadFog();
    }

    var saveTimeout;
    function saveFog(immediate) {
      clearTimeout(saveTimeout);
      if (immediate) {
        canvas.toBlob(function (blob) {
          fetch("/dm/maps/" + mapId + "/fog/progress", {
            method: "PUT",
            body: blob,
          });
        }, "image/png");
      } else {
        saveTimeout = setTimeout(function () {
          canvas.toBlob(function (blob) {
            fetch("/dm/maps/" + mapId + "/fog/progress", {
              method: "PUT",
              body: blob,
            });
          }, "image/png");
        }, 100);
      }
    }

    function loadFog() {
      var img = new Image();
      img.onload = function () {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(img, 0, 0);
      };
      img.onerror = function () {
        // Bug 4A: No progress file exists — fill black and flush save immediately
        ctx.fillStyle = "rgba(0, 0, 0, 1)";
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        saveFog(true);
      };
      img.src = "/dm/maps/" + mapId + "/fog/progress?t=" + Date.now();
    }

    function canvasCoords(e) {
      var rect = canvas.getBoundingClientRect();
      return {
        x: (e.clientX - rect.left) * (canvas.width / rect.width),
        y: (e.clientY - rect.top) * (canvas.height / rect.height),
      };
    }

    function drawBrush(x, y) {
      ctx.globalCompositeOperation =
        tool === "reveal" ? "destination-out" : "source-over";
      ctx.beginPath();
      ctx.arc(x, y, brushSize, 0, Math.PI * 2);
      if (tool === "shroud") ctx.fillStyle = "rgba(0, 0, 0, 1)";
      ctx.fill();
    }

    function drawRect(x1, y1, x2, y2) {
      ctx.globalCompositeOperation =
        tool === "reveal" ? "destination-out" : "source-over";
      if (tool === "shroud") ctx.fillStyle = "rgba(0, 0, 0, 1)";
      ctx.fillRect(
        Math.min(x1, x2), Math.min(y1, y2),
        Math.abs(x2 - x1), Math.abs(y2 - y1)
      );
    }

    canvas.addEventListener("mousedown", function (e) {
      if (e.button !== 0) return;
      drawing = true;
      var pos = canvasCoords(e);
      if (shape === "brush") {
        // Bug 1: initialize lastBrushPos on mousedown
        lastBrushPos = pos;
        drawBrush(pos.x, pos.y);
      } else {
        rectStart = pos;
        fogSnapshot = ctx.getImageData(0, 0, canvas.width, canvas.height);
      }
    });

    canvas.addEventListener("mousemove", function (e) {
      if (!drawing) return;
      var pos = canvasCoords(e);
      if (shape === "brush") {
        // Bug 1: interpolate between lastBrushPos and pos to fill gaps on fast movement
        if (lastBrushPos) {
          var dx = pos.x - lastBrushPos.x;
          var dy = pos.y - lastBrushPos.y;
          var dist = Math.sqrt(dx * dx + dy * dy);
          var steps = Math.ceil(dist / (brushSize / 2));
          for (var s = 0; s <= steps; s++) {
            var t = steps === 0 ? 0 : s / steps;
            drawBrush(
              lastBrushPos.x + dx * t,
              lastBrushPos.y + dy * t
            );
          }
        } else {
          drawBrush(pos.x, pos.y);
        }
        lastBrushPos = pos;
      } else if (rectStart && fogSnapshot) {
        ctx.putImageData(fogSnapshot, 0, 0);
        drawRect(rectStart.x, rectStart.y, pos.x, pos.y);
      }
    });

    canvas.addEventListener("mouseup", function (e) {
      if (!drawing) return;
      drawing = false;
      lastBrushPos = null;
      if (shape === "rect" && rectStart) {
        if (fogSnapshot) ctx.putImageData(fogSnapshot, 0, 0);
        var pos = canvasCoords(e);
        drawRect(rectStart.x, rectStart.y, pos.x, pos.y);
        rectStart = null;
        fogSnapshot = null;
      }
      saveFog();
    });

    canvas.addEventListener("mouseleave", function () {
      if (drawing) {
        drawing = false;
        lastBrushPos = null;
        if (shape === "brush") saveFog();
      }
    });

    // Init token layer on top of fog
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, true);

    // Zoom/pan
    createZoomController(container.querySelector(".canvas-wrap"));

    // --- Toolbar event handling ---
    var fogTools = container.querySelector(".fog-tools");
    var tokenTools = container.querySelector(".token-tools");

    container.addEventListener("click", function (e) {
      // Mode toggle
      var modeBtn = e.target.closest("[data-dm-mode]");
      if (modeBtn) {
        var mode = modeBtn.dataset.dmMode;
        container.querySelectorAll("[data-dm-mode]").forEach(function (b) {
          b.classList.toggle("active", b === modeBtn);
        });
        tokenCtrl.setMode(mode);
        canvas.style.pointerEvents = mode === "fog" ? "auto" : "none";
        if (fogTools) fogTools.style.display = mode === "fog" ? "" : "none";
        if (tokenTools) tokenTools.style.display = mode === "tokens" ? "" : "none";
        return;
      }

      var btn = e.target.closest("[data-fog-tool]");
      if (btn) {
        tool = btn.dataset.fogTool;
        container.querySelectorAll("[data-fog-tool]").forEach(function (b) {
          b.classList.toggle("active", b === btn);
        });
      }
      var shapeBtn = e.target.closest("[data-fog-shape]");
      if (shapeBtn) {
        shape = shapeBtn.dataset.fogShape;
        container.querySelectorAll("[data-fog-shape]").forEach(function (b) {
          b.classList.toggle("active", b === shapeBtn);
        });
      }
      // Bug 4B: Flush any pending debounced save before pushing to players
      if (e.target.closest("[data-fog-push]")) {
        clearTimeout(saveTimeout);
        canvas.toBlob(function (blob) {
          fetch("/dm/maps/" + mapId + "/fog/progress", { method: "PUT", body: blob })
            .then(function () {
              return fetch("/dm/maps/" + mapId + "/fog/push", { method: "POST" });
            });
        }, "image/png");
      }
      if (e.target.closest("[data-fog-clear]")) {
        ctx.globalCompositeOperation = "source-over";
        ctx.fillStyle = "rgba(0, 0, 0, 1)";
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        saveFog();
      }
      if (e.target.closest("[data-fog-reveal-all]")) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        saveFog();
      }
    });

    var slider = container.querySelector("[data-fog-brush-size]");
    if (slider) {
      slider.addEventListener("input", function () {
        brushSize = parseInt(this.value, 10);
      });
    }

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);
    // Bug 2: guard resize listener with alive check
    window.addEventListener("resize", function () {
      if (!alive) return;
      resize();
    });

    return { refreshTokens: tokenCtrl.refresh };
  }

  // --- Player Fog Canvas ---
  function initPlayerCanvas(container, mapId) {
    var mapImg = container.querySelector(".map-image");
    if (!mapImg) return;

    // Bug 2: Remove any existing fog/token canvases left over from a previous init
    var zoomWrap = container.querySelector(".zoom-wrap");
    var oldFog = zoomWrap.querySelector(".fog-canvas");
    if (oldFog) oldFog.remove();
    var oldToken = zoomWrap.querySelector(".token-canvas");
    if (oldToken) oldToken.remove();

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    zoomWrap.appendChild(canvas);
    var ctx = canvas.getContext("2d");

    // Bug 2: alive flag — set false when container is cleaned up by HTMX
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      // Bug 3: Pre-fill black before async fog load to prevent revealed flash
      ctx.fillStyle = "rgba(0,0,0,1)";
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      loadFog();
    }

    function loadFog() {
      var img = new Image();
      img.onload = function () {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(img, 0, 0);
      };
      img.onerror = function () {
        ctx.fillStyle = "rgba(0, 0, 0, 1)";
        ctx.fillRect(0, 0, canvas.width, canvas.height);
      };
      img.src = "/maps/" + mapId + "/fog?t=" + Date.now();
    }

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);
    // Bug 2: guard resize listener with alive check
    window.addEventListener("resize", function () {
      if (!alive) return;
      resize();
    });

    // Token layer on top (players can always interact with moveable tokens)
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, false);

    // Zoom/pan
    createZoomController(container.querySelector(".canvas-wrap"));

    return {
      refresh: loadFog,
      refreshTokens: tokenCtrl.refresh,
    };
  }

  window.DungeonRevealer = {
    initDMCanvas: initDMCanvas,
    initPlayerCanvas: initPlayerCanvas,
  };
})();
