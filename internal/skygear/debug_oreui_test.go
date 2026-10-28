package skygear

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOreResHandlerAnswersProbeFiles(t *testing.T) {
	h := (&oreResServers{}).handler(19132, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for path, want := range map[string]string{
		"/probe.js?t=http-same-port\"x": `window.__ouiRes("http-same-portx")`,
		"/probe.css":                    ".oui-res-probe{width:13px}",
		"/probe.html?t=a<b>":            `<div class="oui-res-html">ab</div>`,
		"/probe.png":                    "\x89PNG",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s: got %q, want %q in it", path, rec.Body.String(), want)
		}
	}
}
