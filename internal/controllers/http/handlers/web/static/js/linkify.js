(function () {
    'use strict';

    // Matches http(s) URLs up to whitespace. Trailing punctuation is trimmed below.
    const URL_RE = /https?:\/\/[^\s]+/g;
    const TRAILING_PUNCT = /[.,;:!?)\]}'"»›]+$/;

    // renderTextWithLinks clears `element` and rebuilds its content as a mix of
    // text nodes and anchor elements — safe against HTML injection because URLs
    // are only inserted via anchor.href / anchor.textContent.
    function renderTextWithLinks(element, text) {
        while (element.firstChild) {
            element.removeChild(element.firstChild);
        }

        if (!text) {
            return;
        }

        let lastIndex = 0;
        let match;

        URL_RE.lastIndex = 0;

        while ((match = URL_RE.exec(text)) !== null) {
            let url = match[0];
            let trail = '';

            const trailMatch = url.match(TRAILING_PUNCT);
            if (trailMatch) {
                trail = trailMatch[0];
                url = url.slice(0, url.length - trail.length);
            }

            const start = match.index;

            if (start > lastIndex) {
                element.appendChild(document.createTextNode(text.slice(lastIndex, start)));
            }

            const anchor = document.createElement('a');
            anchor.href = url;
            anchor.textContent = url;
            anchor.className = 'message-link';
            anchor.target = '_blank';
            anchor.rel = 'noopener noreferrer';
            element.appendChild(anchor);

            if (trail) {
                element.appendChild(document.createTextNode(trail));
            }

            lastIndex = start + match[0].length;
        }

        if (lastIndex < text.length) {
            element.appendChild(document.createTextNode(text.slice(lastIndex)));
        }
    }

    window.renderTextWithLinks = renderTextWithLinks;
})();
