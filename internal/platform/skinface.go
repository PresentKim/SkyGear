package platform

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/google/uuid"
)

// skinFace is the front of the head of s as 8x8 colours "rrggbb" row by row,
// the hat layer drawn over the face, or nil when s has no face to show (no
// pixels, a face all clear). The face is where the skin's geometry puts the
// front of its head bone (Persona and custom models), else where the
// standard humanoid skin has it.
func skinFace(s skin.Skin) []string {
	b := s.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || len(s.Pix) < b.Dx()*b.Dy()*4 {
		return nil
	}
	face, hat := faceRects(s)
	out := make([]string, 0, 64)
	seen := false
	for y := range 8 {
		for x := range 8 {
			r, g, bl, a := sample(s, face, x, y)
			if hr, hg, hb, ha := sample(s, hat, x, y); ha >= 128 {
				r, g, bl, a = hr, hg, hb, 255
			}
			if a < 128 {
				out = append(out, "")
				continue
			}
			seen = true
			out = append(out, fmt.Sprintf("%02x%02x%02x", r, g, bl))
		}
	}
	if !seen {
		return nil
	}
	return out
}

// rect is a face of a cube on the skin texture, in texture pixels.
type rect struct{ x, y, w, h float64 }

// sample is the colour at the face cell (x, y) of 8x8 in r on s (clear when
// r is empty or off the texture).
func sample(s skin.Skin, r rect, x, y int) (uint8, uint8, uint8, uint8) {
	if r.w <= 0 || r.h <= 0 {
		return 0, 0, 0, 0
	}
	b := s.Bounds()
	px, py := int(r.x+(float64(x)+0.5)*r.w/8), int(r.y+(float64(y)+0.5)*r.h/8)
	if px < 0 || py < 0 || px >= b.Dx() || py >= b.Dy() {
		return 0, 0, 0, 0
	}
	i := (py*b.Dx() + px) * 4
	return s.Pix[i], s.Pix[i+1], s.Pix[i+2], s.Pix[i+3]
}

// faceRects are the front of the head and of the hat on the texture of s.
func faceRects(s skin.Skin) (face, hat rect) {
	scale := float64(s.Bounds().Dx()) / 64
	face = rect{8 * scale, 8 * scale, 8 * scale, 8 * scale}
	hat = rect{40 * scale, 8 * scale, 8 * scale, 8 * scale}
	if f, h, ok := modelFace(s); ok {
		return f, h
	}
	return face, hat
}

// geometry is the part of a skin model the head is read from: the format
// 1.12 one (minecraft:geometry) and the older one (geometry.* keys).
type geometry struct {
	Description struct {
		TextureWidth  float64 `json:"texture_width"`
		TextureHeight float64 `json:"texture_height"`
	} `json:"description"`
	TextureWidth  float64 `json:"texturewidth"`
	TextureHeight float64 `json:"textureheight"`
	Bones         []struct {
		Name  string `json:"name"`
		Cubes []struct {
			Size [3]float64      `json:"size"`
			UV   json.RawMessage `json:"uv"`
		} `json:"cubes"`
	} `json:"bones"`
}

// modelFace finds the head and hat faces in the model of s: the front face
// of the first cube of the bones "head" and "hat", with box UV.
func modelFace(s skin.Skin) (face, hat rect, ok bool) {
	if len(s.Model) == 0 {
		return
	}
	var geos []geometry
	var doc struct {
		Geometry []geometry `json:"minecraft:geometry"`
	}
	if json.Unmarshal(s.Model, &doc) == nil && len(doc.Geometry) > 0 {
		geos = doc.Geometry
	} else {
		var old map[string]json.RawMessage
		if json.Unmarshal(s.Model, &old) != nil {
			return
		}
		for k, v := range old {
			var g geometry
			if k != "format_version" && json.Unmarshal(v, &g) == nil && len(g.Bones) > 0 {
				geos = append(geos, g)
			}
		}
	}
	for _, g := range geos {
		tw := g.Description.TextureWidth
		if tw == 0 {
			tw = g.TextureWidth
		}
		if tw == 0 {
			tw = 64
		}
		scale := float64(s.Bounds().Dx()) / tw
		find := func(name string) (rect, bool) {
			for _, b := range g.Bones {
				if b.Name != name || len(b.Cubes) == 0 {
					continue
				}
				c := b.Cubes[0]
				var uv [2]float64
				if json.Unmarshal(c.UV, &uv) != nil {
					return rect{}, false
				}
				w, h, d := c.Size[0], c.Size[1], c.Size[2]
				return rect{(uv[0] + d) * scale, (uv[1] + d) * scale, w * scale, h * scale}, w > 0 && h > 0
			}
			return rect{}, false
		}
		if f, found := find("head"); found {
			h, _ := find("hat")
			return f, h, true
		}
	}
	return
}

// faces caches skinFace by player and skin, as reading a Persona model is
// not free and the main screen is drawn often.
var faces = struct {
	sync.Mutex
	m map[uuid.UUID]cachedFace
}{m: map[uuid.UUID]cachedFace{}}

type cachedFace struct {
	skin string
	face []string
}

// faceOf is skinFace of p's skin.
func faceOf(p *player.Player) []string {
	s := p.Skin()
	id := s.FullID
	faces.Lock()
	c, ok := faces.m[p.UUID()]
	faces.Unlock()
	if ok && c.skin == id && id != "" {
		return c.face
	}
	f := skinFace(s)
	faces.Lock()
	faces.m[p.UUID()] = cachedFace{skin: id, face: f}
	faces.Unlock()
	return f
}

// forgetFace drops p's cached face (p left the server).
func forgetFace(p *player.Player) {
	faces.Lock()
	delete(faces.m, p.UUID())
	faces.Unlock()
}
