// Keeps the status bar in step with HTMX: the active nav pill follows the
// swapped page, and the connection pill reports polling state.
(function () {
  const WAITING_DELAY_MS = 600;

  function markActiveNav() {
    const path = window.location.pathname;
    document.querySelectorAll(".nav a").forEach(function (link) {
      const active = new URL(link.href, window.location.origin).pathname === path;
      link.classList.toggle("on", active);
      if (active) {
        link.setAttribute("aria-current", "page");
      } else {
        link.removeAttribute("aria-current");
      }
    });
  }

  function connectionPill() {
    const indicator = document.getElementById("conn");
    if (!indicator) return;

    const label = indicator.querySelector("span");
    let pending = 0;
    let waitingTimer = null;

    function set(state, text) {
      indicator.dataset.state = state;
      if (label) label.textContent = text;
    }

    function clearWaitingTimer() {
      if (waitingTimer !== null) {
        clearTimeout(waitingTimer);
        waitingTimer = null;
      }
    }

    // A 2s poll that answers in milliseconds should not flash the pill. Only a
    // request still outstanding after WAITING_DELAY_MS is worth reporting.
    document.body.addEventListener("htmx:beforeRequest", function () {
      pending++;
      if (waitingTimer !== null || indicator.dataset.state === "offline") return;
      waitingTimer = setTimeout(function () {
        waitingTimer = null;
        if (pending > 0) set("waiting", "polling");
      }, WAITING_DELAY_MS);
    });

    document.body.addEventListener("htmx:afterRequest", function (event) {
      pending = Math.max(0, pending - 1);
      if (pending > 0) return;
      clearWaitingTimer();
      set(event.detail.successful ? "live" : "offline",
          event.detail.successful ? "live" : "disconnected");
    });

    document.body.addEventListener("htmx:sendError", function () {
      pending = 0;
      clearWaitingTimer();
      set("offline", "disconnected");
    });

    document.body.addEventListener("htmx:timeout", function () {
      pending = 0;
      clearWaitingTimer();
      set("offline", "timed out");
    });
  }

  function trackScroll() {
    const bar = document.querySelector(".topbar");
    if (!bar) return;
    const update = function () {
      bar.classList.toggle("scrolled", window.scrollY > 4);
    };
    window.addEventListener("scroll", update, { passive: true });
    update();
  }

  document.body.addEventListener("htmx:afterSettle", markActiveNav);
  window.addEventListener("popstate", markActiveNav);
  markActiveNav();
  connectionPill();
  trackScroll();
})();
