// Fog-of-War Canvas + Token Layer + Wall Layer + Dynamic Lighting
(function () {
  "use strict";

  // --- Zoom/Pan Controller ---
  function createZoomController(canvasWrap) {
    var zoomWrap = canvasWrap.querySelector(".zoom-wrap");
    if (!zoomWrap) return { isSpaceDown: function() { return false; } };

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
    var spaceDown = false;
    var spacePanning = false;

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

    // Middle-mouse drag or Space+left-drag to pan
    canvasWrap.addEventListener("mousedown", function (e) {
      if (e.button === 1) {
        e.preventDefault();
        panning = true;
        panStartX = e.clientX;
        panStartY = e.clientY;
        lastPanX = panX;
        lastPanY = panY;
      } else if (e.button === 0 && spaceDown) {
        e.preventDefault();
        panning = true;
        spacePanning = true;
        panStartX = e.clientX;
        panStartY = e.clientY;
        lastPanX = panX;
        lastPanY = panY;
        canvasWrap.style.cursor = "grabbing";
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
      if (e.button === 0 && spacePanning) {
        panning = false;
        spacePanning = false;
        canvasWrap.style.cursor = spaceDown ? "grab" : "";
      }
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

    document.addEventListener("keydown", function (e) {
      if (e.code === "Space" && !e.repeat
          && document.activeElement.tagName !== "INPUT"
          && document.activeElement.tagName !== "TEXTAREA") {
        e.preventDefault();
        spaceDown = true;
        canvasWrap.style.cursor = "grab";
      }
    });

    document.addEventListener("keyup", function (e) {
      if (e.code === "Space") {
        spaceDown = false;
        if (spacePanning) {
          panning = false;
          spacePanning = false;
        }
        canvasWrap.style.cursor = "";
      }
    });

    return { isSpaceDown: function() { return spaceDown; } };
  }

  // --- Visibility helpers for dynamic lighting ---

  // Returns t (ray parameter at intersection) or null.
  // Ray: (ox,oy) + t*(dx,dy); Segment: (x1,y1)+(x2-x1,y2-y1)*s for s in [0,1]
  function raySegmentIntersect(ox, oy, dx, dy, x1, y1, x2, y2) {
    var ex = x2 - x1, ey = y2 - y1;
    var det = ex * dy - dx * ey;
    if (Math.abs(det) < 1e-10) return null;
    var rx = x1 - ox, ry = y1 - oy;
    var t = (ex * ry - ey * rx) / det;
    var s = (dx * ry - dy * rx) / det;
    if (t >= 0 && s >= 0 && s <= 1) return t;
    return null;
  }

  // Expand polygon walls to {x1,y1,x2,y2} edge segments for the visibility algorithm.
  function getWallSegments(walls) {
    var segs = [];
    walls.forEach(function (w) {
      if (!w.points || w.points.length < 2) return;
      for (var i = 0; i < w.points.length; i++) {
        var p1 = w.points[i];
        var p2 = w.points[(i + 1) % w.points.length];
        segs.push({ x1: p1.x, y1: p1.y, x2: p2.x, y2: p2.y });
      }
    });
    return segs;
  }

  // True if the line segment (wall.x1,y1)-(wall.x2,y2) intersects or is inside the circle.
  function segmentIntersectsCircle(cx, cy, r, wall) {
    var dx = wall.x2 - wall.x1, dy = wall.y2 - wall.y1;
    var fx = wall.x1 - cx, fy = wall.y1 - cy;
    var a = dx * dx + dy * dy;
    if (a < 1e-10) {
      return fx * fx + fy * fy <= r * r;
    }
    var b = 2 * (fx * dx + fy * dy);
    var c = fx * fx + fy * fy - r * r;
    var disc = b * b - 4 * a * c;
    if (disc < 0) return false;
    var sqrtDisc = Math.sqrt(disc);
    var t1 = (-b - sqrtDisc) / (2 * a);
    var t2 = (-b + sqrtDisc) / (2 * a);
    return (t1 >= 0 && t1 <= 1) || (t2 >= 0 && t2 <= 1) || (t1 < 0 && t2 > 1);
  }

  // Minimum distance from pos to wall segment.
  function distToSegment(pos, wall) {
    var dx = wall.x2 - wall.x1, dy = wall.y2 - wall.y1;
    var len2 = dx * dx + dy * dy;
    if (len2 < 1e-10) {
      var ex = pos.x - wall.x1, ey = pos.y - wall.y1;
      return Math.sqrt(ex * ex + ey * ey);
    }
    var t = ((pos.x - wall.x1) * dx + (pos.y - wall.y1) * dy) / len2;
    t = Math.max(0, Math.min(1, t));
    var nx = wall.x1 + t * dx - pos.x;
    var ny = wall.y1 + t * dy - pos.y;
    return Math.sqrt(nx * nx + ny * ny);
  }

  // Compute visibility polygon using endpoint-visibility (Amit Patel) algorithm.
  // Returns array of {x, y, angle} sorted by angle.
  function computeVisibilityPolygon(ox, oy, sightRadius, walls) {
    var eps = 0.0001;

    // Expand polygon walls to edge segments, then filter to sight range
    var activeWalls = getWallSegments(walls).filter(function (w) {
      return segmentIntersectsCircle(ox, oy, sightRadius, w);
    });

    // Start with evenly-spaced angles around the full circle for a smooth circular boundary
    var angles = [];
    var circleSteps = 72; // one sample every 5 degrees
    for (var i = 0; i < circleSteps; i++) {
      angles.push((i / circleSteps) * 2 * Math.PI - Math.PI);
    }

    // Add tight triples around each wall endpoint to get sharp shadow edges
    activeWalls.forEach(function (w) {
      var a1 = Math.atan2(w.y1 - oy, w.x1 - ox);
      var a2 = Math.atan2(w.y2 - oy, w.x2 - ox);
      angles.push(a1 - eps, a1, a1 + eps, a2 - eps, a2, a2 + eps);
    });

    // For each angle cast a ray; default distance is sightRadius (circle boundary)
    var points = [];
    angles.forEach(function (angle) {
      var dx = Math.cos(angle);
      var dy = Math.sin(angle);
      var minT = sightRadius;
      activeWalls.forEach(function (w) {
        var t = raySegmentIntersect(ox, oy, dx, dy, w.x1, w.y1, w.x2, w.y2);
        if (t !== null && t < minT) minT = t;
      });
      points.push({ x: ox + dx * minT, y: oy + dy * minT, angle: angle });
    });

    points.sort(function (a, b) { return a.angle - b.angle; });
    return points;
  }

  // --- Token Layer (shared between DM and Player) ---
  function createTokenLayer(container, mapImg, mapId, isDM, isSpaceDown, onTokensLoaded) {
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

    // alive flag — set false when container is removed by HTMX
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
        if (t.shape === "square") {
          ctx.rect(t.x - t.radius, t.y - t.radius, t.radius * 2, t.radius * 2);
        } else {
          ctx.arc(t.x, t.y, t.radius, 0, Math.PI * 2);
        }
        ctx.fillStyle = t.color || "#e94560";
        ctx.globalAlpha = alpha;
        ctx.fill();
        ctx.strokeStyle = "#fff";
        ctx.lineWidth = 2;
        ctx.stroke();
        if (t.label) {
          var fontSize = (t.labelSize > 0) ? t.labelSize : Math.max(12, t.radius * 0.6);
          ctx.fillStyle = "#fff";
          ctx.font = fontSize + "px sans-serif";
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
          if (onTokensLoaded) onTokensLoaded(tokens);
        });
    }

    function hitTest(pos) {
      for (var i = tokens.length - 1; i >= 0; i--) {
        var t = tokens[i];
        if (!isDM && !t.visible) continue;
        if (t.shape === "square") {
          if (pos.x >= t.x - t.radius && pos.x <= t.x + t.radius &&
              pos.y >= t.y - t.radius && pos.y <= t.y + t.radius) return t;
        } else {
          var dx = pos.x - t.x;
          var dy = pos.y - t.y;
          if (dx * dx + dy * dy <= t.radius * t.radius) return t;
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
      if (isSpaceDown && isSpaceDown()) return;
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
      body.set("shape", getToolbarValue("data-token-shape", ""));
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

      var shapeCircleSelected = (!token.shape || token.shape === "circle") ? " selected" : "";
      var shapeSquareSelected = token.shape === "square" ? " selected" : "";
      var labelSizeVal = token.labelSize > 0 ? token.labelSize : 0;
      var sightRadiusVal = token.sightRadius > 0 ? token.sightRadius : parseFloat(container.dataset.gridSize || "50") * 5;

      popup.innerHTML =
        '<label>Name <input type="text" data-field="label" value="' + (token.label || "").replace(/"/g, "&quot;") + '" class="token-input-text"/></label>' +
        '<label>Color <input type="color" data-field="color" value="' + (token.color || "#e94560") + '"/></label>' +
        '<label>Size <input type="range" data-field="radius" min="5" max="300" value="' + (token.radius || 20) + '"/></label>' +
        '<label>Shape <select data-field="shape"><option value=""' + shapeCircleSelected + '>Circle</option><option value="square"' + shapeSquareSelected + '>Square</option></select></label>' +
        '<label>Label px <input type="range" data-field="labelSize" min="0" max="60" value="' + labelSizeVal + '"/> <span class="token-popup-hint">(0=auto)</span></label>' +
        '<label>Sight Radius <input type="range" data-field="sightRadius" min="0" max="2000" value="' + sightRadiusVal + '"/> <span class="token-popup-hint">(0=off)</span></label>' +
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
        body.set("shape", popup.querySelector('[data-field="shape"]').value);
        body.set("labelSize", popup.querySelector('[data-field="labelSize"]').value);
        body.set("sightRadius", popup.querySelector('[data-field="sightRadius"]').value);
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

      var confirmingDelete = false;
      var deleteBtn = popup.querySelector("[data-popup-delete]");
      deleteBtn.addEventListener("click", function () {
        if (!confirmingDelete) {
          confirmingDelete = true;
          deleteBtn.textContent = "Confirm Delete?";
          deleteBtn.style.background = "#c0392b";
          setTimeout(function () {
            if (confirmingDelete) {
              confirmingDelete = false;
              deleteBtn.textContent = "Delete";
              deleteBtn.style.background = "";
            }
          }, 3000);
          return;
        }
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

    // Guard SSE and resize listeners with alive check
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

  // --- Wall Layer (DM only) — polygon drawing ---
  function createWallLayer(container, mapImg, mapId, isSpaceDownFn) {
    var zoomWrap = container.querySelector(".zoom-wrap");
    var canvas = document.createElement("canvas");
    canvas.className = "wall-canvas";
    var cursorCanvas = zoomWrap.querySelector(".cursor-canvas");
    if (cursorCanvas) {
      zoomWrap.insertBefore(canvas, cursorCanvas);
    } else {
      zoomWrap.appendChild(canvas);
    }
    var ctx = canvas.getContext("2d");

    var walls = [];          // committed polygons from server
    var currentPoly = [];   // vertices of the polygon being drawn
    var mousePos = null;
    var alive = true;
    var SNAP_DIST = 12;      // px in canvas space to snap/close polygon
    var dragState = null;       // {wall, idx} while dragging a vertex
    var dragMoved = false;      // true once mouse moved during a drag
    var suppressNextClick = false; // set in mouseup after any vertex drag to block click
    var contextMenu = null;    // floating context menu div

    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

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

    function dist2(a, b) {
      var dx = a.x - b.x, dy = a.y - b.y;
      return dx * dx + dy * dy;
    }

    function render() {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.setLineDash([]);

      // Draw committed polygon walls
      walls.forEach(function (w) {
        if (!w.points || w.points.length < 2) return;
        ctx.beginPath();
        ctx.moveTo(w.points[0].x, w.points[0].y);
        for (var i = 1; i < w.points.length; i++) {
          ctx.lineTo(w.points[i].x, w.points[i].y);
        }
        ctx.closePath();
        ctx.fillStyle = "rgba(255, 140, 0, 0.12)";
        ctx.fill();
        ctx.strokeStyle = "rgba(255, 140, 0, 0.9)";
        ctx.lineWidth = 2;
        ctx.stroke();
        // Vertex dots
        ctx.fillStyle = "rgba(255, 140, 0, 0.9)";
        w.points.forEach(function (p) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, 4, 0, Math.PI * 2);
          ctx.fill();
        });
      });

      // Draw in-progress polygon
      if (currentPoly.length > 0) {
        // Edges so far
        ctx.beginPath();
        ctx.moveTo(currentPoly[0].x, currentPoly[0].y);
        for (var i = 1; i < currentPoly.length; i++) {
          ctx.lineTo(currentPoly[i].x, currentPoly[i].y);
        }
        ctx.strokeStyle = "rgba(255, 200, 0, 0.9)";
        ctx.lineWidth = 2;
        ctx.setLineDash([]);
        ctx.stroke();

        // Dashed preview to mouse
        if (mousePos) {
          ctx.setLineDash([6, 4]);
          ctx.beginPath();
          ctx.moveTo(currentPoly[currentPoly.length - 1].x, currentPoly[currentPoly.length - 1].y);
          ctx.lineTo(mousePos.x, mousePos.y);
          ctx.stroke();
          ctx.setLineDash([]);
        }

        // Vertex dots
        ctx.fillStyle = "rgba(255, 200, 0, 0.9)";
        currentPoly.forEach(function (p, idx) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, idx === 0 ? 6 : 4, 0, Math.PI * 2);
          ctx.fill();
        });

        // Highlight first vertex when close enough to close
        if (mousePos && currentPoly.length >= 2 && dist2(mousePos, currentPoly[0]) < SNAP_DIST * SNAP_DIST) {
          ctx.beginPath();
          ctx.arc(currentPoly[0].x, currentPoly[0].y, 8, 0, Math.PI * 2);
          ctx.strokeStyle = "rgba(255, 255, 255, 0.9)";
          ctx.lineWidth = 2;
          ctx.stroke();
        }
      }
    }

    function closePoly() {
      if (currentPoly.length < 3) return;
      var pts = currentPoly.map(function (p) { return { x: p.x, y: p.y }; });
      currentPoly = [];
      mousePos = null;
      fetch("/dm/maps/" + mapId + "/walls", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ points: pts }),
      }).then(function () { loadWalls(); });
    }

    function loadWalls() {
      fetch("/maps/" + mapId + "/walls")
        .then(function (r) { return r.json(); })
        .then(function (data) {
          walls = data || [];
          render();
        });
    }

    function setActive(active) {
      canvas.style.pointerEvents = active ? "auto" : "none";
      if (!active) {
        currentPoly = [];
        mousePos = null;
        render();
      }
    }

    function findVertexHit(pos, radius) {
      for (var wi = 0; wi < walls.length; wi++) {
        var w = walls[wi];
        if (!w.points) continue;
        for (var vi = 0; vi < w.points.length; vi++) {
          if (dist2(pos, w.points[vi]) <= radius * radius) {
            return { wall: w, idx: vi };
          }
        }
      }
      return null;
    }

    function findWallNearEdge(pos, threshold) {
      var hitWall = null;
      var hitDist = Infinity;
      walls.forEach(function (w) {
        if (!w.points || w.points.length < 2) return;
        for (var i = 0; i < w.points.length; i++) {
          var seg = {
            x1: w.points[i].x, y1: w.points[i].y,
            x2: w.points[(i + 1) % w.points.length].x,
            y2: w.points[(i + 1) % w.points.length].y,
          };
          var d = distToSegment(pos, seg);
          if (d < threshold && d < hitDist) {
            hitDist = d;
            hitWall = w;
          }
        }
      });
      return hitWall;
    }

    function showWallContextMenu(clientX, clientY, wallId) {
      hideWallContextMenu();
      var menu = document.createElement("div");
      menu.style.cssText = "position:fixed;z-index:9999;background:#1a1a1a;border:1px solid #555;border-radius:4px;padding:4px 0;";
      menu.style.left = clientX + "px";
      menu.style.top = clientY + "px";
      var btn = document.createElement("button");
      btn.textContent = "Delete wall";
      btn.style.cssText = "display:block;width:100%;padding:6px 14px;background:none;border:none;color:#f55;cursor:pointer;text-align:left;font-size:13px;white-space:nowrap;";
      btn.addEventListener("mouseenter", function () { btn.style.background = "#2a2a2a"; });
      btn.addEventListener("mouseleave", function () { btn.style.background = "none"; });
      btn.addEventListener("click", function () {
        hideWallContextMenu();
        fetch("/dm/maps/" + mapId + "/walls/" + wallId, { method: "DELETE" })
          .then(function () { loadWalls(); });
      });
      menu.appendChild(btn);
      document.body.appendChild(menu);
      contextMenu = menu;
      setTimeout(function () {
        document.addEventListener("click", hideWallContextMenu, { once: true });
      }, 0);
    }

    function hideWallContextMenu() {
      if (contextMenu) {
        contextMenu.remove();
        contextMenu = null;
      }
    }

    canvas.addEventListener("contextmenu", function (e) {
      e.preventDefault();
      if (currentPoly.length > 0) {
        currentPoly = [];
        mousePos = null;
        render();
      }
      var pos = coords(e);
      var hitWall = findWallNearEdge(pos, 10);
      if (hitWall) {
        showWallContextMenu(e.clientX, e.clientY, hitWall.id);
      }
    });

    canvas.addEventListener("pointerdown", function (e) {
      if (e.button !== 0) return;
      if (isSpaceDownFn && isSpaceDownFn()) return;
      if (currentPoly.length > 0) return;
      var pos = coords(e);
      var hit = findVertexHit(pos, 8);
      if (hit) {
        dragState = hit;
        dragMoved = false;
        canvas.style.cursor = "grabbing";
        canvas.setPointerCapture(e.pointerId);
      }
    });

    canvas.addEventListener("pointermove", function (e) {
      if (dragState) {
        var pos = coords(e);
        dragState.wall.points[dragState.idx] = pos;
        dragMoved = true;
        render();
      } else {
        mousePos = coords(e);
        var hit = findVertexHit(mousePos, 8);
        canvas.style.cursor = hit ? "grab" : "crosshair";
        render();
      }
    });

    canvas.addEventListener("pointerup", function (e) {
      if (!alive) return;
      if (dragState) {
        suppressNextClick = true;
        if (dragMoved) {
          var wall = dragState.wall;
          fetch("/dm/maps/" + mapId + "/walls/" + wall.id, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ points: wall.points }),
          }).then(function () { loadWalls(); });
        }
        dragState = null;
        dragMoved = false;
        canvas.style.cursor = "crosshair";
      }
    });

    canvas.addEventListener("click", function (e) {
      if (isSpaceDownFn && isSpaceDownFn()) return;
      if (suppressNextClick) {
        suppressNextClick = false;
        return;
      }
      var pos = coords(e);
      if (currentPoly.length === 0) {
        // Start a new polygon
        currentPoly = [pos];
      } else {
        // Close polygon if clicking near first vertex
        if (currentPoly.length >= 2 && dist2(pos, currentPoly[0]) < SNAP_DIST * SNAP_DIST) {
          closePoly();
        } else {
          currentPoly.push(pos);
        }
      }
      render();
    });

    canvas.addEventListener("dblclick", function (e) {
      if (currentPoly.length >= 3) {
        // Remove the extra point added by the preceding click event
        currentPoly.pop();
        closePoly();
      }
    });

    document.addEventListener("keydown", function (e) {
      if (!alive) return;
      if (e.key === "Escape" && currentPoly.length > 0) {
        currentPoly = [];
        mousePos = null;
        render();
      }
      if ((e.key === "Enter") && currentPoly.length >= 3) {
        closePoly();
      }
    });

    document.body.addEventListener("sse:wallUpdate", function () {
      if (!alive) return;
      loadWalls();
    });

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);
    loadWalls();

    return { setActive: setActive, resize: resize, refresh: loadWalls };
  }

  // --- Dynamic Lighting Layer (Player only) ---
  function createDynamicLightingLayer(container, mapImg, mapId) {
    var zoomWrap = container.querySelector(".zoom-wrap");
    var canvas = document.createElement("canvas");
    canvas.className = "lighting-canvas";
    zoomWrap.appendChild(canvas);
    var ctx = canvas.getContext("2d");

    var walls = [];
    var tokens = [];
    var enabled = false;
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      render();
    }

    function render() {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      if (!enabled) return;

      var lightingTokens = tokens.filter(function (t) {
        return t.visible && t.sightRadius > 0;
      });

      // Fill black overlay
      ctx.globalCompositeOperation = "source-over";
      ctx.fillStyle = "rgba(0,0,0,1)";
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      if (lightingTokens.length === 0) return;

      // Cut out visibility polygon for each token with sight radius
      ctx.globalCompositeOperation = "destination-out";
      lightingTokens.forEach(function (t) {
        var poly = computeVisibilityPolygon(t.x, t.y, t.sightRadius, walls);
        if (poly.length < 3) return;
        ctx.beginPath();
        ctx.moveTo(poly[0].x, poly[0].y);
        for (var i = 1; i < poly.length; i++) {
          ctx.lineTo(poly[i].x, poly[i].y);
        }
        ctx.closePath();
        ctx.fill();
      });

      ctx.globalCompositeOperation = "source-over";
    }

    function loadWalls(id) {
      fetch("/maps/" + (id || mapId) + "/walls")
        .then(function (r) { return r.json(); })
        .then(function (data) {
          walls = data || [];
          render();
        });
    }

    function loadSettings(id) {
      fetch("/maps/" + (id || mapId) + "/settings")
        .then(function (r) { return r.json(); })
        .then(function (data) {
          enabled = data.dynamicLighting || false;
          render();
        });
    }

    function setTokens(newTokens) {
      tokens = newTokens || [];
      render();
    }

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);

    return { resize: resize, render: render, setTokens: setTokens, loadWalls: loadWalls, loadSettings: loadSettings };
  }

  // --- DM Fog Canvas ---
  function initDMCanvas(container, mapId) {
    var mapImg = container.querySelector(".map-image");
    if (!mapImg) return;

    // Remove any existing canvases left over from a previous init
    var zoomWrap = container.querySelector(".zoom-wrap");
    var oldFog = zoomWrap.querySelector(".fog-canvas");
    if (oldFog) oldFog.remove();
    var oldToken = zoomWrap.querySelector(".token-canvas");
    if (oldToken) oldToken.remove();
    var oldCursor = zoomWrap.querySelector(".cursor-canvas");
    if (oldCursor) oldCursor.remove();
    var oldWall = zoomWrap.querySelector(".wall-canvas");
    if (oldWall) oldWall.remove();
    var oldGrid = zoomWrap.querySelector(".grid-canvas");
    if (oldGrid) oldGrid.remove();

    // Grid canvas sits below the fog layer
    var gridCanvas = document.createElement("canvas");
    gridCanvas.className = "grid-canvas";
    zoomWrap.appendChild(gridCanvas);
    var gridCtx = gridCanvas.getContext("2d");

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    zoomWrap.appendChild(canvas);
    var ctx = canvas.getContext("2d");

    // Brush cursor indicator canvas
    var cursorCanvas = document.createElement("canvas");
    cursorCanvas.className = "cursor-canvas";
    zoomWrap.appendChild(cursorCanvas);
    var cursorCtx = cursorCanvas.getContext("2d");

    var tool = "reveal";
    var shape = "brush";
    var gridSize = parseFloat(container.dataset.gridSize || "50");
    var gridOffsetX = parseFloat(container.dataset.gridOffsetX || "0");
    var gridOffsetY = parseFloat(container.dataset.gridOffsetY || "0");
    var gridEnabled = container.dataset.gridEnabled === "true";
    var brushSize = gridSize;
    var drawing = false;
    var rectStart = null;
    var fogSnapshot = null;
    var lastBrushPos = null;
    var lastMousePos = null;

    // alive flag — set false when container is cleaned up by HTMX
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    function renderGrid() {
      gridCtx.clearRect(0, 0, gridCanvas.width, gridCanvas.height);
      if (!gridEnabled || gridSize <= 0) return;
      gridCtx.strokeStyle = "rgba(255,255,255,0.25)";
      gridCtx.lineWidth = 1;
      gridCtx.beginPath();
      var startX = ((gridOffsetX % gridSize) + gridSize) % gridSize;
      var startY = ((gridOffsetY % gridSize) + gridSize) % gridSize;
      for (var x = startX; x <= gridCanvas.width; x += gridSize) {
        gridCtx.moveTo(x, 0); gridCtx.lineTo(x, gridCanvas.height);
      }
      for (var y = startY; y <= gridCanvas.height; y += gridSize) {
        gridCtx.moveTo(0, y); gridCtx.lineTo(gridCanvas.width, y);
      }
      gridCtx.stroke();
    }

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      cursorCanvas.width = mapImg.naturalWidth;
      cursorCanvas.height = mapImg.naturalHeight;
      cursorCanvas.style.width = mapImg.clientWidth + "px";
      cursorCanvas.style.height = mapImg.clientHeight + "px";
      gridCanvas.width = mapImg.naturalWidth;
      gridCanvas.height = mapImg.naturalHeight;
      gridCanvas.style.width = mapImg.clientWidth + "px";
      gridCanvas.style.height = mapImg.clientHeight + "px";
      renderGrid();
      if (wallCtrl) wallCtrl.resize();
      loadFog();
    }

    var lastCursorX = -1, lastCursorY = -1;

    function drawCursorAt(x, y) {
      var pad = brushSize + 2;
      if (lastCursorX >= 0) {
        cursorCtx.clearRect(lastCursorX - pad, lastCursorY - pad, pad * 2, pad * 2);
      } else {
        cursorCtx.clearRect(0, 0, cursorCanvas.width, cursorCanvas.height);
      }
      lastCursorX = x; lastCursorY = y;
      cursorCtx.beginPath();
      cursorCtx.arc(x, y, brushSize, 0, Math.PI * 2);
      cursorCtx.strokeStyle = "rgba(255,255,255,0.8)";
      cursorCtx.lineWidth = 2;
      cursorCtx.stroke();
    }

    function clearCursor() {
      cursorCtx.clearRect(0, 0, cursorCanvas.width, cursorCanvas.height);
      lastCursorX = -1; lastCursorY = -1;
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
        // No progress file exists — fill black and flush save immediately
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

    // Zoom/pan (init first so isSpaceDown is available)
    var zoomCtrl = createZoomController(container.querySelector(".canvas-wrap"));
    var canvasWrapEl = container.querySelector(".canvas-wrap");

    document.addEventListener("keydown", function (e) {
      if (e.code === "Space" && !e.repeat
          && document.activeElement.tagName !== "INPUT"
          && document.activeElement.tagName !== "TEXTAREA") {
        clearCursor();
        canvas.style.cursor = "";
      }
    });
    document.addEventListener("keyup", function (e) {
      if (e.code === "Space") {
        if (shape === "brush" && lastMousePos) {
          drawCursorAt(lastMousePos.x, lastMousePos.y);
          canvas.style.cursor = "none";
          canvasWrapEl.style.cursor = "none";
        } else {
          canvas.style.cursor = "crosshair";
          canvasWrapEl.style.cursor = "crosshair";
        }
      }
    });

    canvas.addEventListener("mousedown", function (e) {
      if (e.button !== 0) return;
      if (zoomCtrl.isSpaceDown()) return;
      drawing = true;
      var pos = canvasCoords(e);
      if (shape === "brush") {
        lastBrushPos = pos;
        drawBrush(pos.x, pos.y);
      } else {
        rectStart = pos;
        fogSnapshot = ctx.getImageData(0, 0, canvas.width, canvas.height);
      }
    });

    canvas.addEventListener("mousemove", function (e) {
      var pos = canvasCoords(e);
      lastMousePos = pos;
      if (shape === "brush") {
        if (zoomCtrl.isSpaceDown()) {
          clearCursor();
          canvas.style.cursor = "";
        } else {
          drawCursorAt(pos.x, pos.y);
        }
      } else {
        clearCursor();
        canvas.style.cursor = zoomCtrl.isSpaceDown() ? "" : "crosshair";
      }
      if (!drawing) return;
      if (shape === "brush") {
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
      clearCursor();
      canvasWrapEl.style.cursor = "";
      if (drawing) {
        drawing = false;
        lastBrushPos = null;
        if (shape === "brush") saveFog();
      }
    });

    // Init token layer on top of fog
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, true, zoomCtrl.isSpaceDown);

    // Init wall layer (inserted before cursor-canvas)
    var wallCtrl = createWallLayer(container, mapImg, mapId, zoomCtrl.isSpaceDown);

    // --- Toolbar event handling ---
    var fogTools = container.querySelector(".fog-tools");
    var tokenTools = container.querySelector(".token-tools");
    var wallTools = container.querySelector(".wall-tools");

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
        cursorCanvas.style.display = mode === "fog" ? "" : "none";
        if (mode !== "fog") canvasWrapEl.style.cursor = "";
        if (fogTools) fogTools.style.display = mode === "fog" ? "" : "none";
        if (tokenTools) tokenTools.style.display = mode === "tokens" ? "" : "none";
        if (wallTools) wallTools.style.display = mode === "walls" ? "" : "none";
        wallCtrl.setActive(mode === "walls");
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
        if (shape === "brush") {
          canvas.style.cursor = "none";
          canvasWrapEl.style.cursor = "none";
        } else {
          clearCursor();
          canvas.style.cursor = "crosshair";
          canvasWrapEl.style.cursor = "crosshair";
        }
      }
      // Flush pending debounced save before pushing to players
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

    // Dynamic lighting toggle
    var dlToggle = container.querySelector("[data-toggle-dynamic-lighting]");
    if (dlToggle) {
      dlToggle.addEventListener("change", function () {
        var body = new URLSearchParams();
        body.set("dynamicLighting", this.checked ? "true" : "false");
        body.set("gridSize", gridSize);
        body.set("gridOffsetX", gridOffsetX);
        body.set("gridOffsetY", gridOffsetY);
        body.set("gridEnabled", gridEnabled);
        fetch("/dm/maps/" + mapId + "/settings", { method: "POST", body: body });
      });
    }

    var slider = container.querySelector("[data-fog-brush-size]");
    if (slider) {
      slider.addEventListener("input", function () {
        brushSize = parseInt(this.value, 10);
      });
    }

    var gridSaveTimeout;
    function saveGridSettings() {
      clearTimeout(gridSaveTimeout);
      gridSaveTimeout = setTimeout(function () {
        var body = new URLSearchParams();
        body.set("gridSize", gridSize);
        body.set("gridOffsetX", gridOffsetX);
        body.set("gridOffsetY", gridOffsetY);
        body.set("gridEnabled", gridEnabled);
        var dlEl = container.querySelector("[data-toggle-dynamic-lighting]");
        body.set("dynamicLighting", dlEl && dlEl.checked ? "true" : "false");
        fetch("/dm/maps/" + mapId + "/settings", { method: "POST", body: body });
      }, 300);
    }

    var gridEnabledEl = container.querySelector("[data-grid-enabled]");
    if (gridEnabledEl) {
      gridEnabledEl.addEventListener("change", function () {
        gridEnabled = this.checked;
        renderGrid();
        saveGridSettings();
      });
    }
    var gridSizeEl = container.querySelector("[data-grid-size]");
    if (gridSizeEl) {
      gridSizeEl.addEventListener("input", function () {
        gridSize = parseFloat(this.value) || 50;
        renderGrid();
        saveGridSettings();
        var brushSlider = container.querySelector("[data-fog-brush-size]");
        if (brushSlider) { brushSlider.value = gridSize; brushSize = gridSize; }
        var tokenSlider = container.querySelector("[data-token-radius]");
        if (tokenSlider) { tokenSlider.value = gridSize / 2; }
      });
    }
    var gridOffsetXEl = container.querySelector("[data-grid-offset-x]");
    if (gridOffsetXEl) {
      gridOffsetXEl.addEventListener("input", function () {
        gridOffsetX = parseFloat(this.value) || 0;
        renderGrid();
        saveGridSettings();
      });
    }
    var gridOffsetYEl = container.querySelector("[data-grid-offset-y]");
    if (gridOffsetYEl) {
      gridOffsetYEl.addEventListener("input", function () {
        gridOffsetY = parseFloat(this.value) || 0;
        renderGrid();
        saveGridSettings();
      });
    }

    if (mapImg.complete) requestAnimationFrame(resize);
    else mapImg.addEventListener("load", resize);
    // Guard resize listener with alive check
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

    // Remove any existing canvases left over from a previous init
    var zoomWrap = container.querySelector(".zoom-wrap");
    var oldFog = zoomWrap.querySelector(".fog-canvas");
    if (oldFog) oldFog.remove();
    var oldToken = zoomWrap.querySelector(".token-canvas");
    if (oldToken) oldToken.remove();
    var oldLighting = zoomWrap.querySelector(".lighting-canvas");
    if (oldLighting) oldLighting.remove();
    var oldGrid = zoomWrap.querySelector(".grid-canvas");
    if (oldGrid) oldGrid.remove();

    // Create lighting layer first (below fog)
    var lightingLayer = createDynamicLightingLayer(container, mapImg, mapId);

    // Grid canvas sits above lighting but below fog
    var gridCanvas = document.createElement("canvas");
    gridCanvas.className = "grid-canvas";
    zoomWrap.appendChild(gridCanvas);
    var gridCtx = gridCanvas.getContext("2d");

    var gridSize = 50;
    var gridOffsetX = 0;
    var gridOffsetY = 0;
    var gridEnabled = false;

    function renderGrid() {
      gridCtx.clearRect(0, 0, gridCanvas.width, gridCanvas.height);
      if (!gridEnabled || gridSize <= 0) return;
      gridCtx.strokeStyle = "rgba(255,255,255,0.25)";
      gridCtx.lineWidth = 1;
      gridCtx.beginPath();
      var startX = ((gridOffsetX % gridSize) + gridSize) % gridSize;
      var startY = ((gridOffsetY % gridSize) + gridSize) % gridSize;
      for (var x = startX; x <= gridCanvas.width; x += gridSize) {
        gridCtx.moveTo(x, 0); gridCtx.lineTo(x, gridCanvas.height);
      }
      for (var y = startY; y <= gridCanvas.height; y += gridSize) {
        gridCtx.moveTo(0, y); gridCtx.lineTo(gridCanvas.width, y);
      }
      gridCtx.stroke();
    }

    function loadGridSettings() {
      fetch("/maps/" + mapId + "/settings")
        .then(function (r) { return r.json(); })
        .then(function (data) {
          gridSize = data.gridSize || 50;
          gridOffsetX = data.gridOffsetX || 0;
          gridOffsetY = data.gridOffsetY || 0;
          gridEnabled = data.gridEnabled || false;
          renderGrid();
        });
    }

    var canvas = document.createElement("canvas");
    canvas.className = "fog-canvas";
    zoomWrap.appendChild(canvas);
    var ctx = canvas.getContext("2d");

    // alive flag — set false when container is cleaned up by HTMX
    var alive = true;
    container.addEventListener("htmx:beforeCleanupElement", function () { alive = false; });

    function resize() {
      canvas.width = mapImg.naturalWidth;
      canvas.height = mapImg.naturalHeight;
      canvas.style.width = mapImg.clientWidth + "px";
      canvas.style.height = mapImg.clientHeight + "px";
      gridCanvas.width = mapImg.naturalWidth;
      gridCanvas.height = mapImg.naturalHeight;
      gridCanvas.style.width = mapImg.clientWidth + "px";
      gridCanvas.style.height = mapImg.clientHeight + "px";
      renderGrid();
      lightingLayer.resize();
      // Pre-fill black before async fog load to prevent revealed flash
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
    // Guard resize listener with alive check
    window.addEventListener("resize", function () {
      if (!alive) return;
      resize();
    });

    // Zoom/pan (init first so isSpaceDown is available for token layer)
    var zoomCtrl = createZoomController(container.querySelector(".canvas-wrap"));

    // Token layer — pass setTokens as onTokensLoaded so lighting updates when tokens change
    var tokenCtrl = createTokenLayer(container, mapImg, mapId, false, zoomCtrl.isSpaceDown, lightingLayer.setTokens);

    // Load initial state after first resize
    if (mapImg.complete) {
      lightingLayer.loadSettings(mapId);
      lightingLayer.loadWalls(mapId);
      loadGridSettings();
    } else {
      mapImg.addEventListener("load", function () {
        lightingLayer.loadSettings(mapId);
        lightingLayer.loadWalls(mapId);
        loadGridSettings();
      }, { once: true });
    }

    return {
      refresh: loadFog,
      refreshTokens: tokenCtrl.refresh,
      refreshWalls: function () { lightingLayer.loadWalls(mapId); },
      refreshSettings: function () {
        lightingLayer.loadSettings(mapId);
        loadGridSettings();
      },
    };
  }

  window.DungeonRevealer = {
    initDMCanvas: initDMCanvas,
    initPlayerCanvas: initPlayerCanvas,
  };
})();
