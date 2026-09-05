// caetano — JS partilhado, vanilla, sem framework/CDN.

(function () {
  "use strict";

  /**
   * Mostra uma mensagem toast temporária no canto inferior direito.
   * @param {string} message
   * @param {number} [durationMs=3000]
   */
  window.caeToast = function caeToast(message, durationMs) {
    durationMs = durationMs || 3000;
    var el = document.createElement("div");
    el.className = "cae-toast";
    el.textContent = message;
    document.body.appendChild(el);
    requestAnimationFrame(function () {
      el.classList.add("cae-toast--visible");
    });
    setTimeout(function () {
      el.classList.remove("cae-toast--visible");
      setTimeout(function () {
        el.remove();
      }, 200);
    }, durationMs);
  };
})();
