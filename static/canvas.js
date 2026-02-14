// Fog-of-War Canvas + Token Layer
(function () {
  "use strict";

  // --- Token Layer (shared between DM and Player) ---
  function createTokenLayer(container, mapImg, mapId, isDM) {
    const canvas = document.createElement("canvas");
    canvas.className = "token-canvas";
    container.querySelector(".canvas-wrap").appendChild(canvas);
    const ctx = canvas.getContext("2d");
    var tokens = [];
    var dragging = null;
    var dragOffset = { x: 0, y: 0 };

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
        ctx.beginPath();
        ctx.arc(t.x, t.y, t.radius, 0, Math.PI * 2);
        ctx.fillStyle = t.color || "#e94560";
        ctx.globalAlpha = !isDM && !t.visible ? 0 : isDM && !t.visible ? 0.4 : 0.8;
        ctx.fill();
        ctx.globalAlpha = 1;
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
        var dx = pos.x - t.x;
        var dy = pos.y - t.y;
        if (dx * dx + dy * dy <= t.radius * t.radius) {
          return t;
        }
      }
      return null;
    }

    canvas.addEventListener("mousedown", function (e) {
      var pos = coords(e);
      var token = hitTest(pos);
      if (token && (isDM || token.moveable)) {
        dragging = token;
        dragOffset.x = token.x - pos.x;
        dragOffset.y = token.y - pos.y;
        e.stopPropagation();
      } else if (isDM) {
        // No token hit — disable token canvas so fog canvas receives all events
        canvas.style.pointerEvents = "none";
        var below = document.elementFromPoint(e.clientX, e.clientY);
        if (below && below !== canvas) {
          below.dispatchEvent(new MouseEvent("mousedown", e));
        }
      }
    });

    // Re-enable token canvas pointer events when fog drawing ends
    if (isDM) {
      document.addEventListener("mouseup", function () {
        canvas.style.pointerEvents = "auto";
      });
    }

    canvas.addEventListener("mousemove", function (e) {
      if (!dragging) return;
      var pos = coords(e);
      dragging.x = pos.x + dragOffset.x;
      dragging.y = pos.y + dragOffset.y;
      render();
    });

    canvas.addEventListener("mouseup", function (e) {
      if (!dragging) return;
      var url = isDM
        ? "/dm/maps/" + mapId + "/tokens/" + dragging.id
        : "/maps/" + mapId + "/tokens/" + dragging.id;
      var body = new URLSearchParams();
      body.set("x", dragging.x);
      body.set("y", dragging.y);
      fetch(url, { method: "PUT", body: body });
      dragging = null;
    });

    // DM: double-click to add token
    if (isDM) {
      canvas.addEventListener("dblclick", function (e) {
        var pos = coords(e);
        var body = new URLSearchParams();
        body.set("x", pos.x);
        body.set("y", pos.y);
        body.set("radius", "20");
        body.set("label", "");
        body.set("color", "#e94560");
        body.set("visible", "true");
        body.set("moveable", "true");
        fetch("/dm/maps/" + mapId + "/tokens", {
          method: "POST",
          body: body,
        }).then(function () { loadTokens(); });
      });
    }

    if (mapImg.complete) resize();
    else mapImg.addEventListener("load", resize);
    window.addEventListener("resize", resize);
    loadTokens();

    return { refresh: loadTokens, resize: resize };
  }

  // --- DM Fog Canvas ---
  function initDMCanvas(container, mapId) {
    var mapImg = container.querySelector(".map-image");
    if (!mapImg) return;

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    container.querySelector(".canvas-wrap").appendChild(canvas);
    var ctx = canvas.getContext("2d");

    var tool = "reveal";
    var shape = "brush";
    var brushSize = 40;
    var drawing = false;
    var rectStart = null;
    var fogSnapshot = null;

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
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
      drawing = true;
      var pos = canvasCoords(e);
      if (shape === "brush") {
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
        drawBrush(pos.x, pos.y);
      } else if (rectStart && fogSnapshot) {
        ctx.putImageData(fogSnapshot, 0, 0);
        drawRect(rectStart.x, rectStart.y, pos.x, pos.y);
      }
    });

    canvas.addEventListener("mouseup", function (e) {
      if (!drawing) return;
      drawing = false;
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
        if (shape === "brush") saveFog();
      }
    });

    var saveTimeout;
    function saveFog() {
      clearTimeout(saveTimeout);
      saveTimeout = setTimeout(function () {
        canvas.toBlob(function (blob) {
          fetch("/dm/maps/" + mapId + "/fog/progress", {
            method: "PUT",
            body: blob,
          });
        }, "image/png");
      }, 100);
    }

    container.addEventListener("click", function (e) {
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
      if (e.target.closest("[data-fog-push]"))
        fetch("/dm/maps/" + mapId + "/fog/push", { method: "POST" });
      if (e.target.closest("[data-fog-clear]")) {
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

    if (mapImg.complete) resize();
    else mapImg.addEventListener("load", resize);
    window.addEventListener("resize", resize);

    // Init token layer on top of fog
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, true);

    return { refreshTokens: tokenCtrl.refresh };
  }

  // --- Player Fog Canvas ---
  function initPlayerCanvas(container, mapId) {
    var mapImg = container.querySelector(".map-image");
    if (!mapImg) return;

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    container.querySelector(".canvas-wrap").appendChild(canvas);
    var ctx = canvas.getContext("2d");

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
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

    if (mapImg.complete) resize();
    else mapImg.addEventListener("load", resize);
    window.addEventListener("resize", resize);

    // Token layer on top
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, false);

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
