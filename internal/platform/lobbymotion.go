package platform

import (
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/dfhook"
)

// The lobby scene is not a still picture (docs/design/01-platform.md §1):
// the camera drifts slowly around the island, and the player's character
// looks about and now and then waves or stretches. Everything is sent to
// that player's client only; the server's own player does not move.

// lobbyGlide is how long the camera takes from one stop to the next; a new
// stop is sent as the camera arrives, so it never rests.
const lobbyGlide = 14 * time.Second

// lobbyOrbit are the camera's stops: the angle around the island from the
// front (degrees, east positive) and the height above the island's middle.
// The first is lobbyCamera.
var lobbyOrbit = []struct{ angle, height float64 }{
	{0, 3}, {11, 3.8}, {4, 3.3}, {-9, 4.1}, {-3, 2.8},
}

// lobbyCameraAt is the camera of stop i: 12 blocks from the island's middle,
// looking at it.
func lobbyCameraAt(i int) (pos mgl64.Vec3, pitch, yaw float64) {
	s := lobbyOrbit[i%len(lobbyOrbit)]
	a := mgl64.DegToRad(s.angle)
	pivot := LobbySpawn.Vec3Middle().Add(mgl64.Vec3{0, 0.5, 0})
	pos = pivot.Add(mgl64.Vec3{12 * math.Sin(a), s.height, 12 * math.Cos(a)})
	return pos, mgl64.RadToDeg(math.Atan2(s.height, 12)), 180 - s.angle
}

// lobbyIdles are the character's small moves (pack animations,
// animations/lobby.animation.json): looking at the tree, the lantern, the
// flowers and the sky. lobbyGestures are the bigger ones, now and then:
// waving at the camera and stretching.
var (
	lobbyIdles    = []string{"animation.df_lobby.look_left", "animation.df_lobby.look_right", "animation.df_lobby.look_down", "animation.df_lobby.look_up"}
	lobbyGestures = []string{"animation.df_lobby.wave", "animation.df_lobby.stretch"}
)

// lobbyMotion is where one player's lobby scene is.
type lobbyMotion struct {
	stop, idle                        int
	nextCamera, nextLook, nextGesture time.Time
}

var lobbyMotions = struct {
	sync.Mutex
	m map[string]*lobbyMotion
}{m: map[string]*lobbyMotion{}}

// startLobbyMotion starts p's scene from the camera's first stop.
func startLobbyMotion(p *player.Player) {
	now := time.Now()
	lobbyMotions.Lock()
	lobbyMotions.m[p.Name()] = &lobbyMotion{
		nextCamera:  now.Add(2 * time.Second),
		nextLook:    now.Add(3 * time.Second),
		nextGesture: now.Add(12 * time.Second),
	}
	lobbyMotions.Unlock()
}

// stopLobbyMotion forgets p's scene (leaving the lobby or the server).
func stopLobbyMotion(p *player.Player) {
	lobbyMotions.Lock()
	delete(lobbyMotions.m, p.Name())
	lobbyMotions.Unlock()
}

// moveLobbyScene moves p's scene on if it is time: called every second for
// the players in the lobby.
func moveLobbyScene(p *player.Player, now time.Time) {
	lobbyMotions.Lock()
	m, ok := lobbyMotions.m[p.Name()]
	lobbyMotions.Unlock()
	if !ok {
		return
	}
	if !now.Before(m.nextCamera) {
		m.stop = (m.stop + 1) % len(lobbyOrbit)
		pos, pitch, yaw := lobbyCameraAt(m.stop)
		dfhook.GlideCamera(p, pos, pitch, yaw, lobbyGlide.Seconds())
		m.nextCamera = now.Add(lobbyGlide)
	}
	if !now.Before(m.nextGesture) {
		dfhook.Animate(p, lobbyGestures[rand.IntN(len(lobbyGestures))])
		m.nextGesture = now.Add(time.Duration(25+rand.IntN(15)) * time.Second)
		m.nextLook = now.Add(5 * time.Second)
		return
	}
	if !now.Before(m.nextLook) {
		next := rand.IntN(len(lobbyIdles) - 1)
		if next >= m.idle {
			next++ // never the same move twice
		}
		m.idle = next
		dfhook.Animate(p, lobbyIdles[next])
		m.nextLook = now.Add(time.Duration(5+rand.IntN(5)) * time.Second)
	}
}
