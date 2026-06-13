(function () {
  var endpoint = '/api/v1/analytics/events';

  function sourceFromLocation() {
    var params = new URLSearchParams(window.location.search);
    return params.get('utm_source') || params.get('ref') || '';
  }

  function send(eventType, target) {
    var payload = JSON.stringify({
      event_type: eventType,
      path: window.location.pathname + window.location.search,
      title: document.title,
      referrer: document.referrer,
      source: sourceFromLocation(),
      locale: document.documentElement.lang || navigator.language || '',
      target: target || ''
    });

    if (navigator.sendBeacon) {
      navigator.sendBeacon(endpoint, new Blob([payload], { type: 'application/json' }));
      return;
    }

    fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payload,
      keepalive: true
    }).catch(function () {});
  }

  send('pageview');

  document.addEventListener('click', function (event) {
    var link = event.target.closest && event.target.closest('a,button');
    if (!link) return;
    var label = link.getAttribute('data-analytics') ||
      link.getAttribute('aria-label') ||
      link.textContent ||
      link.getAttribute('href') ||
      '';
    label = label.replace(/\s+/g, ' ').trim().slice(0, 120);
    if (label) send('click', label);
  }, true);
})();
