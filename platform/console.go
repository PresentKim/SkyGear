package platform

import (
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/instance"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The lobby has no chat window: the main screen is always open, and no
// engine command of the Ore UI page opens the game's chat
// (docs/tech/oreui-probes.md). Operators type commands in a console screen
// instead (the main screen's command button, the system menu): what a
// command sends back (command output and chat lines) is shown in its log
// while the console is open.

// consoleLines is how many lines the console keeps.
const consoleLines = 80

// consoleRefresh is how long the console waits for more lines before it is
// drawn again with what came.
const consoleRefresh = 80 * time.Millisecond

// console is one operator's console: the log, whether it is open, and the
// redraw waiting for lines that came.
type console struct {
	lines   []string
	open    bool
	pf      *Platform
	h       *world.EntityHandle
	pending bool
}

var consoles = struct {
	sync.Mutex
	m map[uuid.UUID]*console
	// byConn finds the console of a connection for the packet hook.
	byConn map[*sghook.Conn]*console
}{m: map[uuid.UUID]*console{}, byConn: map[*sghook.Conn]*console{}}

func init() {
	sghook.OnServerPacket(func(c *sghook.Conn, pk packet.Packet) bool {
		var lines []string
		switch pk := pk.(type) {
		case *packet.Text:
			lines = append(lines, textLine(pk.Message, pk.Parameters, pk.TextType == packet.TextTypeTranslation || pk.NeedsTranslation))
		case *packet.CommandOutput:
			for _, m := range pk.OutputMessages {
				line := textLine(m.Message, m.Parameters, true)
				if !m.Success {
					line = "§c" + line
				}
				lines = append(lines, line)
			}
		default:
			return false
		}
		consoles.Lock()
		con := consoles.byConn[c]
		if con == nil || !con.open {
			consoles.Unlock()
			return false
		}
		con.add(lines...)
		redraw := !con.pending
		con.pending = true
		h, pf := con.h, con.pf
		consoles.Unlock()
		if redraw {
			h.DoAfter(consoleRefresh, func(_ *world.Tx, e world.Entity) {
				consoles.Lock()
				con.pending = false
				open := con.open
				consoles.Unlock()
				if open {
					pf.showConsole(e.(*player.Player))
				}
			})
		}
		return false
	})
	sghook.OnClose(func(c *sghook.Conn) {
		consoles.Lock()
		delete(consoles.byConn, c)
		consoles.Unlock()
	})
}

// textLine is a line of the log for a message: a translation key (with its
// parameters after it) is drawn in the client's language.
func textLine(msg string, params []string, translate bool) string {
	key := strings.TrimPrefix(msg, "%")
	if translate && key != "" && !strings.ContainsAny(key, " §") {
		fallback := key
		if len(params) > 0 {
			fallback += " " + strings.Join(params, " ")
		}
		line := i18n.ClientKey(key, fallback)
		if len(params) > 0 {
			line += " §7" + strings.Join(params, " ")
		}
		return line
	}
	return msg
}

// add appends lines to the log, keeping the last consoleLines. consoles
// must be locked.
func (c *console) add(lines ...string) {
	for _, l := range lines {
		for _, part := range strings.Split(l, "\n") {
			if part != "" {
				c.lines = append(c.lines, part)
			}
		}
	}
	if n := len(c.lines) - consoleLines; n > 0 {
		c.lines = append([]string(nil), c.lines[n:]...)
	}
}

// showConsole shows operator p the console.
func (pf *Platform) showConsole(p *player.Player) {
	if pf.isOperator == nil || !pf.isOperator(p) {
		return
	}
	lang := i18n.Of(p)
	consoles.Lock()
	con := consoles.m[p.UUID()]
	if con == nil {
		con = &console{pf: pf}
		consoles.m[p.UUID()] = con
	}
	con.open, con.h = true, p.H()
	if c := sghook.Of(p); c != nil {
		consoles.byConn[c] = con
	}
	lines := append([]string(nil), con.lines...)
	consoles.Unlock()
	if len(lines) == 0 {
		lines = []string{i18n.T(lang, "platform.console.empty")}
	}
	ui.ConsoleScreen(p, i18n.T(lang, "platform.console.title"), ui.Console{
		Lines:     lines,
		Hint:      i18n.T(lang, "platform.console.hint"),
		Submit:    i18n.T(lang, "platform.console.run"),
		Run:       pf.runConsole,
		BackLabel: i18n.T(lang, "platform.saves.lobby"),
		Back:      func(p *player.Player, _ *world.Tx) { pf.showLobbyMenu(p) },
	})
}

// runConsole runs the command operator p typed in the console and shows the
// console again; the output comes in through the packet hook.
func (pf *Platform) runConsole(p *player.Player, _ *world.Tx, line string) {
	if pf.isOperator == nil || !pf.isOperator(p) {
		return
	}
	line = strings.TrimSpace(line)
	if line == "" {
		pf.showConsole(p)
		return
	}
	if !strings.HasPrefix(line, "/") {
		line = "/" + line
	}
	consoles.Lock()
	if con := consoles.m[p.UUID()]; con != nil {
		con.add("§e> " + line)
	}
	consoles.Unlock()
	p.ExecuteCommand(line)
	if pf.m.InLobby(instance.AccountID(p)) {
		pf.showConsole(p)
	}
}

// closeConsole stops showing what comes back in p's console (another screen
// took its place).
func closeConsole(p *player.Player) {
	consoles.Lock()
	if con := consoles.m[p.UUID()]; con != nil {
		con.open = false
	}
	consoles.Unlock()
}

// forgetConsole drops p's console (p left the server).
func forgetConsole(p *player.Player) {
	consoles.Lock()
	delete(consoles.m, p.UUID())
	consoles.Unlock()
}
