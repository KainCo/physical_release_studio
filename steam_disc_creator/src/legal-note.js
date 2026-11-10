/* Rechtlicher Hinweis (nur privater Gebrauch / Rechte bei den Inhabern) – wird am Seitenende eingeblendet.
   Sprache: localStorage 'prs_lang' (wie die Studios), sonst <html lang>. Wird beim Drucken ausgeblendet. */
(function () {
  var T = {
    de: 'Nur für den privaten Gebrauch. Alle Rechte an Spielen, Covern, Verpackungen, Namen und Marken liegen bei den jeweiligen Inhabern. Die Vorlagen und Tools sind nicht dazu gedacht, geschützte Werke zu verkaufen oder zu verbreiten. Dies ist ein unabhängiges, nicht kommerzielles Hobbyprojekt ohne Verbindung zu den Inhabern von Marken wie GOG oder Steam. Nutzer sind für die von ihnen verwendeten Bilder und Inhalte selbst verantwortlich. Alle Daten werden nur lokal in deinem Browser verarbeitet und nicht hochgeladen.',
    en: 'For private use only. All rights to games, cover art, packaging, names and trademarks belong to their respective owners. The templates and tools are not intended for selling or distributing protected works. This is an independent, non-commercial hobby project not affiliated with the owners of trademarks such as GOG or Steam. Users are responsible for the images and content they use. All data is processed locally in your browser only and is not uploaded.'
  };
  function lang() {
    var fixed = document.documentElement.getAttribute('data-note-lang');   // Seiten ohne Sprachumschalter (z. B. Datenschutz) legen die Sprache fest
    if (fixed) return fixed === 'en' ? 'en' : 'de';
    var l = null;
    try { l = localStorage.getItem('prs_lang'); } catch (e) {}
    l = l || document.documentElement.lang || 'de';
    return l.slice(0, 2) === 'en' ? 'en' : 'de';
  }
  function init() {
    if (document.documentElement.hasAttribute('data-note-off')) return;   // Hauptseite zeigt den Hinweis im Footer
    var st = document.createElement('style');
    st.textContent = '.prs-legal-note{box-sizing:border-box;max-width:min(1000px,calc(100% - 24px));margin:24px auto 16px;padding:10px 16px;text-align:center;font:11px/1.5 Inter,system-ui,sans-serif;color:#94a3b8;background:rgba(9,13,22,.88);border-radius:8px}' +
      '@media print{.prs-legal-note{display:none!important}}';
    document.head.appendChild(st);
    var el = document.createElement('p');
    el.className = 'prs-legal-note';
    el.setAttribute('role', 'note');
    function render() { el.textContent = T[lang()]; }
    render();
    document.body.appendChild(el);
    new MutationObserver(render).observe(document.documentElement, { attributes: true, attributeFilter: ['lang'] });
    document.addEventListener('click', function (e) {
      if (e.target.closest && e.target.closest('#lang-en,#lang-de')) setTimeout(render, 0);
    });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init); else init();
})();
