package skygear

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
)

// Ore UI probes (/sgdebug oreinput, oreres): DDUI screens that try in the
// client what the Ore UI page can do. The page's reports are shown on the
// screen and written to the server log (docs/tech/oreui-probes.md).

// oreTraced holds, by connection, what takes the packets the client sends
// while a probe screen is open.
var oreTraced sync.Map

func init() {
	sghook.OnClientPacket(func(c *sghook.Conn, pk packet.Packet) bool {
		if v, ok := oreTraced.Load(c); ok {
			v.(func(packet.Packet))(pk)
		}
		return false
	})
}

// oreQuiet are the packets a client sends all the time, left out of a trace.
func oreQuiet(pk packet.Packet) bool {
	switch pk.(type) {
	case *packet.PlayerAuthInput, *packet.NetworkStackLatency, *packet.SubChunkRequest, *packet.Animate:
		return true
	}
	return false
}

// oreProbe is what a probe screen of one player saw: the lines the page
// reported and the packets the client sent.
type oreProbe struct {
	mu      sync.Mutex
	reports int
	last    string
	packets map[string]int
	scripts []string
	// requests are the last HTTP requests the resource probe's servers got.
	requests []string
	// show draws info on the player's screen.
	show func()
}

// request notes an HTTP request of the resource probe.
func (pr *oreProbe) request(line string) {
	pr.mu.Lock()
	pr.requests = append(pr.requests, line)
	if len(pr.requests) > 8 {
		pr.requests = pr.requests[1:]
	}
	pr.mu.Unlock()
	pr.show()
}

func (pr *oreProbe) info() string {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	var kinds []string
	for _, k := range slices.Sorted(maps.Keys(pr.packets)) {
		kinds = append(kinds, fmt.Sprintf("%s x%d", k, pr.packets[k]))
	}
	lines := []string{fmt.Sprintf("§eserver got§r %d: %s", pr.reports, pr.last), "§epackets§r " + strings.Join(kinds, ", ")}
	for _, s := range pr.scripts {
		lines = append(lines, "§eScriptMessage§r "+s)
	}
	for _, s := range pr.requests {
		lines = append(lines, "§ehttp§r "+s)
	}
	return strings.Join(lines, "\n")
}

// openProbe shows nodes in a probe screen for p, tracing the packets of p's
// client until it closes. closed, if not nil, also runs then.
func (g *Game) openProbe(p *player.Player, title string, nodes []any, closed func()) *oreProbe {
	pr := &oreProbe{packets: map[string]int{}}
	h := p.H()
	show := func() {
		h.Do(func(_ *world.Tx, e world.Entity) { ui.SetProbeInfo(e.(*player.Player), pr.info()) })
	}
	pr.show = show
	conn := sghook.Of(p)
	if conn != nil {
		oreTraced.Store(conn, func(pk packet.Packet) {
			if oreQuiet(pk) {
				return
			}
			kind := strings.TrimPrefix(fmt.Sprintf("%T", pk), "*packet.")
			pr.mu.Lock()
			pr.packets[kind]++
			if sm, ok := pk.(*packet.ScriptMessage); ok {
				pr.scripts = append(pr.scripts, fmt.Sprintf("%s %q", sm.Identifier, sm.Data))
			}
			pr.mu.Unlock()
			if sm, ok := pk.(*packet.ScriptMessage); ok {
				g.log.Info("ore ui probe script message", "player", p.Name(), "identifier", sm.Identifier, "data", string(sm.Data))
			}
			if _, ok := pk.(*packet.ServerBoundDataStore); !ok {
				show()
			}
		})
	}
	report := func(p *player.Player, _ *world.Tx, id, line string) {
		g.log.Info("ore ui probe", "probe", title, "player", p.Name(), "id", id, "line", line)
		pr.mu.Lock()
		pr.reports++
		pr.last = line
		pr.mu.Unlock()
		ui.SetProbeInfo(p, pr.info())
	}
	ui.ProbeScreen(p, title, nodes, report, func(*player.Player, *world.Tx) {
		if conn != nil {
			oreTraced.Delete(conn)
		}
		if closed != nil {
			closed()
		}
	})
	ui.SetProbeInfo(p, pr.info())
	return pr
}

// oreUIOnly tells whether p sees DDUI screens, writing an error to o if not.
func oreUIOnly(p *player.Player, o *cmd.Output) bool {
	if !ui.OreUIFor(p) {
		o.Errorf("Ore UI screens are off for %s (/uimode, [ResourcePack] Encrypt)", p.Name())
		return false
	}
	return true
}

type debugOreInput struct {
	adminBase
	Sub cmd.SubCommand `cmd:"oreinput"`
}

// Run opens a screen that lists the input events the Ore UI page gets
// (mouse, wheel, keys, touches, pointer, focus, text input, gamepads), sends
// each to the server and shows what the server got back, with the packets the
// client sent meanwhile (a ScriptMessage among them is shown in full).
func (c debugOreInput) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok || !oreUIOnly(p, o) {
		return
	}
	c.g.openProbe(p, "/sgdebug oreinput", []any{
		ui.ProbeNode("input", "id", "probe-input", "class", "mc-input", "placeholder", "keys, IME"),
		ui.ProbeNode("box", "class", "mc-gap-3"),
		ui.ProbeNode("probe"),
	}, nil)
}

type debugOreRes struct {
	adminBase
	Sub cmd.SubCommand `cmd:"oreres"`
}

// Run opens a screen that tries every way the server could give the Ore UI
// page JS, CSS and HTML: text in the data store, the pack's files and files
// from HTTP servers on the host the client joined, on the game's port and the
// next one (TCP). The servers run until the screen closes and show the
// requests they get.
func (c debugOreRes) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok || !oreUIOnly(p, o) {
		return
	}
	conn := sghook.Of(p)
	if conn == nil {
		o.Errorf("%s has no hooked connection", p.Name())
		return
	}
	host, portText, err := net.SplitHostPort(conn.ClientData().ServerAddress)
	port, perr := strconv.Atoi(portText)
	if err != nil || perr != nil {
		o.Errorf("server address %q: %v %v", conn.ClientData().ServerAddress, err, perr)
		return
	}
	var bases []any
	var failed []string
	for i, name := range []string{"http-same-port", "http-next-port"} {
		if err := oreRes.serve(port+i, c.g.log); err != nil {
			failed = append(failed, err.Error())
		}
		bases = append(bases, map[string]any{"name": name, "base": "http://" + net.JoinHostPort(host, strconv.Itoa(port+i))})
	}
	pr := c.g.openProbe(p, "/sgdebug oreres", []any{ui.ProbeNode("restest",
		"js", `window.__ouiRes("packet");`,
		"css", ".oui-res-probe{width:13px}",
		"html", `<div class="oui-res-html">packet</div>`,
		"http", bases,
	)}, oreRes.stop)
	oreRes.watch(pr)
	for _, f := range failed {
		pr.request(f)
	}
}

// oreRes holds the HTTP servers of /sgdebug oreres by port and the probe
// that shows their requests.
var oreRes = &oreResServers{servers: map[int]*http.Server{}}

type oreResServers struct {
	mu      sync.Mutex
	servers map[int]*http.Server
	probe   *oreProbe
}

func (s *oreResServers) watch(pr *oreProbe) {
	s.mu.Lock()
	s.probe = pr
	s.mu.Unlock()
}

// serve starts the server on TCP port, if it is not running.
func (s *oreResServers) serve(port int, log *slog.Logger) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.servers[port] != nil {
		return nil
	}
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: s.handler(port, log), ReadHeaderTimeout: 5 * time.Second}
	s.servers[port] = srv
	go func() { _ = srv.Serve(l) }()
	return nil
}

// stop closes every server.
func (s *oreResServers) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for port, srv := range s.servers {
		_ = srv.Close()
		delete(s.servers, port)
	}
	s.probe = nil
}

// handler answers the restest node's requests: probe.js calls __ouiRes with
// the tag t, probe.css sizes .oui-res-probe, probe.html holds oui-res-html,
// probe.png is a small picture.
func (s *oreResServers) handler(port int, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		line := fmt.Sprintf("%d %s %s origin=%q referer=%q ua=%q", port, r.Method, r.URL.RequestURI(), r.Header.Get("Origin"), r.Referer(), r.UserAgent())
		log.Info("ore ui probe http", "remote", r.RemoteAddr, "request", line)
		s.mu.Lock()
		pr := s.probe
		s.mu.Unlock()
		if pr != nil {
			pr.request(line)
		}
		tag := oreResTag(r.URL.Query().Get("t"))
		w.Header().Set("Access-Control-Allow-Origin", "*")
		switch r.URL.Path {
		case "/probe.js":
			w.Header().Set("Content-Type", "text/javascript")
			_, _ = fmt.Fprintf(w, "if (window.__ouiRes) window.__ouiRes(%q);\n", tag)
		case "/probe.css":
			w.Header().Set("Content-Type", "text/css")
			_, _ = io.WriteString(w, ".oui-res-probe{width:13px}\n")
		case "/probe.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<div class=\"oui-res-html\">%s</div>\n", tag)
		case "/probe.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(oreResPNG)
		default:
			http.NotFound(w, r)
		}
	})
}

// oreResTag keeps the letters, digits and dashes of t.
func oreResTag(t string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, t)
}

var oreResPNG = func() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 255, B: 255, A: 255}), image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}()
