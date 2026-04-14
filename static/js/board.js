(function () {
  'use strict';

  var CHROME_EDGE_KEY = 'imgbrd.chromeEdge';
  var CHROME_WIDE_MQ = '(min-width: 900px)';
  var CHROME_NARROW_SIDE_MQ = '(max-width: 719px)';
  var lastPostNo = null;
  var refreshTimer = null;
  var chromeResizeObs = null;

  function prefersReducedMotion() {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function normalizeEdgeForViewport(edge) {
    try {
      if (window.matchMedia(CHROME_NARROW_SIDE_MQ).matches && (edge === 'left' || edge === 'right')) {
        return 'top';
      }
    } catch (e) { /* ignore */ }
    return edge;
  }

  function closestTextarea(from) {
    var form = from && from.closest && from.closest('.post-form');
    if (!form) return null;
    return form.querySelector('textarea.md-input');
  }

  function wrapSelection(ta, before, after) {
    if (!ta) return;
    var s = ta.selectionStart;
    var e = ta.selectionEnd;
    var v = ta.value;
    var sel = v.slice(s, e);
    ta.value = v.slice(0, s) + before + sel + after + v.slice(e);
    ta.focus();
    var ns = s + before.length;
    ta.setSelectionRange(ns, ns + sel.length);
  }

  function insertLinePrefix(ta, prefix) {
    if (!ta) return;
    var s = ta.selectionStart;
    var v = ta.value;
    var lineStart = v.lastIndexOf('\n', s - 1) + 1;
    ta.value = v.slice(0, lineStart) + prefix + v.slice(lineStart);
    ta.focus();
    var pos = s + prefix.length;
    ta.setSelectionRange(pos, pos);
  }

  function insertAtCursor(ta, text) {
    if (!ta) return;
    var s = ta.selectionStart;
    var v = ta.value;
    ta.value = v.slice(0, s) + text + v.slice(s);
    ta.focus();
    var pos = s + text.length;
    ta.setSelectionRange(pos, pos);
  }

  function syncChromePadding() {
    var dock = document.getElementById('chrome-dock');
    var b = document.body;
    if (!dock) return;
    b.style.paddingTop = '';
    b.style.paddingBottom = '';
    b.style.paddingLeft = '';
    b.style.paddingRight = '';
    var edge = dock.getAttribute('data-edge') || 'top';
    var gap = 16;
    window.requestAnimationFrame(function () {
      var r = dock.getBoundingClientRect();
      if (edge === 'top') b.style.paddingTop = Math.ceil(r.height + gap) + 'px';
      else if (edge === 'bottom') b.style.paddingBottom = Math.ceil(r.height + gap) + 'px';
      else if (edge === 'left') b.style.paddingLeft = Math.ceil(r.width + gap) + 'px';
      else if (edge === 'right') b.style.paddingRight = Math.ceil(r.width + gap) + 'px';
    });
  }

  function applyChromeEdge(edge) {
    var ok = { top: 1, bottom: 1, left: 1, right: 1 };
    if (!ok[edge]) edge = 'top';
    var requested = edge;
    var displayEdge = normalizeEdgeForViewport(requested);
    document.body.classList.remove(
      'chrome-edge-top',
      'chrome-edge-bottom',
      'chrome-edge-left',
      'chrome-edge-right'
    );
    document.body.classList.add('chrome-edge-' + displayEdge);
    var dock = document.getElementById('chrome-dock');
    if (dock) dock.setAttribute('data-edge', displayEdge);
    try {
      localStorage.setItem(CHROME_EDGE_KEY, requested);
    } catch (e) { /* ignore */ }
    document.querySelectorAll('.chrome-snap-btn').forEach(function (btn) {
      btn.classList.toggle('is-active', btn.getAttribute('data-chrome-edge') === displayEdge);
    });
    syncChromePadding();
  }

  function syncChromeSecondaryDetails() {
    var el = document.getElementById('chrome-secondary-details');
    var sum = document.querySelector('.chrome-secondary-summary');
    if (!el) return;
    var wide = false;
    try {
      wide = window.matchMedia(CHROME_WIDE_MQ).matches;
    } catch (e) {
      wide = window.innerWidth >= 900;
    }
    if (wide) {
      el.classList.add('chrome-secondary--desktop');
      el.open = true;
      if (sum) {
        sum.setAttribute('tabindex', '-1');
        sum.setAttribute('aria-hidden', 'true');
      }
    } else {
      el.classList.remove('chrome-secondary--desktop');
      el.open = false;
      if (sum) {
        sum.removeAttribute('tabindex');
        sum.removeAttribute('aria-hidden');
      }
    }
    syncChromePadding();
  }

  function initChromeDock() {
    var dock = document.getElementById('chrome-dock');
    if (!dock) return;
    var saved;
    try {
      saved = localStorage.getItem(CHROME_EDGE_KEY);
    } catch (e) {
      saved = null;
    }
    applyChromeEdge(saved || 'top');

    document.addEventListener('click', function (ev) {
      var btn = ev.target && ev.target.closest && ev.target.closest('.chrome-snap-btn');
      if (!btn) return;
      var edge = btn.getAttribute('data-chrome-edge');
      if (!edge) return;
      ev.preventDefault();
      applyChromeEdge(edge);
    });

    var details = document.getElementById('chrome-secondary-details');
    if (details) {
      details.addEventListener('toggle', function () {
        syncChromePadding();
      });
    }

    function onViewportChromeResize() {
      var savedEdge;
      try {
        savedEdge = localStorage.getItem(CHROME_EDGE_KEY) || 'top';
      } catch (e) {
        savedEdge = 'top';
      }
      applyChromeEdge(savedEdge);
      syncChromeSecondaryDetails();
      syncChromePadding();
    }

    window.addEventListener('resize', onViewportChromeResize);

    try {
      var mq = window.matchMedia(CHROME_WIDE_MQ);
      if (mq.addEventListener) {
        mq.addEventListener('change', function () {
          onViewportChromeResize();
        });
      } else if (mq.addListener) {
        mq.addListener(onViewportChromeResize);
      }
    } catch (e) { /* ignore */ }

    syncChromeSecondaryDetails();
    if (typeof ResizeObserver !== 'undefined') {
      if (chromeResizeObs) chromeResizeObs.disconnect();
      chromeResizeObs = new ResizeObserver(syncChromePadding);
      chromeResizeObs.observe(dock);
    }
  }

  function setAutorefresh(on) {
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }
    if (!on || prefersReducedMotion()) return;
    refreshTimer = setInterval(function () {
      if (document.hidden) return;
      location.reload();
    }, 25000);
  }

  function escapeHtml(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
  }

  function buildPageTools() {
    var inner = document.getElementById('chrome-page-tools');
    var block = document.getElementById('chrome-tools-block');
    if (!inner) return;

    var thread = document.querySelector('.thread-page[data-thread-id]');
    var board = document.querySelector('.board-page[data-board-slug]');
    var parts = [];

    if (thread) {
      var tid = thread.getAttribute('data-thread-id');
      parts.push('<span class="dock-label">тред</span>');
      parts.push('<a href="/thread/export?id=' + encodeURIComponent(tid) + '">.zip</a>');
      parts.push('<span class="dock-sep" aria-hidden="true">·</span>');
      parts.push('<a href="/api/thread?id=' + encodeURIComponent(tid) + '">JSON</a>');
      parts.push('<span class="dock-sep" aria-hidden="true">·</span>');
      if (prefersReducedMotion()) {
        parts.push('<span class="dock-motion-hint" title="Автообновление отключено из‑за настройки «меньше анимации» в системе">автообновление недоступно</span>');
      } else {
        parts.push('<label class="dock-autorefresh"><input type="checkbox" id="autorefresh"> авто 25с</label>');
      }
    } else if (board) {
      var bs = board.getAttribute('data-board-slug');
      if (bs) {
        parts.push('<span class="dock-label">/' + escapeHtml(bs) + '/</span>');
        parts.push('<a href="/rss/board/' + encodeURIComponent(bs) + '">RSS</a>');
        parts.push('<span class="dock-sep" aria-hidden="true">·</span>');
        parts.push('<a href="/api/board/' + encodeURIComponent(bs) + '">JSON</a>');
      }
    }

    inner.innerHTML = parts.join(' ');

    if (block) {
      block.classList.toggle('chrome-tools--empty', parts.length === 0);
    }

    var cb = document.getElementById('autorefresh');
    if (cb) {
      cb.addEventListener('change', function () {
        setAutorefresh(cb.checked);
      });
    }

    syncChromePadding();
  }

  document.addEventListener('click', function (ev) {
    var t = ev.target;
    if (t.classList && t.classList.contains('post-num')) {
      var post = t.closest('.post');
      if (post && post.dataset.postNo) {
        lastPostNo = post.dataset.postNo;
      }
    }

    if (t.classList && t.classList.contains('md-btn')) {
      var ta = closestTextarea(t);
      if (!ta) return;
      ev.preventDefault();
      if (t.classList.contains('md-code')) {
        wrapSelection(ta, '`', '`');
        return;
      }
      if (t.classList.contains('md-line')) {
        var p = t.getAttribute('data-prefix') || '> ';
        insertLinePrefix(ta, p);
        return;
      }
      if (t.dataset.quote !== undefined) {
        if (!lastPostNo) return;
        insertAtCursor(ta, '>>' + lastPostNo + '\n');
        return;
      }
      var w = t.getAttribute('data-wrap');
      if (w) {
        wrapSelection(ta, w, w);
      }
    }

    if (t.classList && t.classList.contains('captcha-reload')) {
      var img = t.parentElement && t.parentElement.querySelector('.captcha-img');
      if (img) {
        var base = img.getAttribute('data-captcha-src') || img.src.split('?')[0];
        img.src = base + '?reload=' + Date.now();
      }
    }
  });

  document.addEventListener('click', function (ev) {
    var t = ev.target;
    if (t.classList && t.classList.contains('img-expand') && t.tagName !== 'VIDEO') {
      t.classList.toggle('expanded');
    }
  });

  document.addEventListener('DOMContentLoaded', function () {
    initChromeDock();
    buildPageTools();
  });
})();
