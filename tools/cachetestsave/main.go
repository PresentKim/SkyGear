// Command cachetestsave adds a save with test chunks to a server's data folder,
// for /cachetest without playing a save first. Run it with the server
// stopped; it prints the new save number.
//
//	go run ./tools/cachetestsave data
//
// Chunks (sub chunk 4 is y 64..79):
//
//	(0,0) and its neighbours  a 3x3x3 dirt island at the origin
//	(2,0), (3,0)              the same full stone sub chunk 4 (other x z)
//	(4,0)                     the same stone, one sub chunk higher (sub chunk 5)
//	(5,0)                     stone sub chunk 4, filled with diamond first
//	(6,0)                     stone sub chunk 4 with one dirt block
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/presentkim/skygear/platform/db"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: cachetestsave <data folder>")
	}
	dir := os.Args[1]
	d, err := db.Open(filepath.Join(dir, "platform.db"))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := d.TouchAccount("cachetest-owner", "", "owner"); err != nil {
		log.Fatal(err)
	}
	s, err := d.CreateSave("cachetest-owner", 1, "cachetest")
	if err != nil {
		log.Fatal(err)
	}
	prov, err := mcdb.Config{}.Open(filepath.Join(dir, "saves", strconv.FormatInt(s.ID, 10), "world"))
	if err != nil {
		log.Fatal(err)
	}
	w := world.Config{Provider: prov, Generator: world.NopGenerator{}}.New()
	fill := func(tx *world.Tx, cx, cz, y0 int, b world.Block) {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				for y := y0; y < y0+16; y++ {
					tx.SetBlock(cube.Pos{cx*16 + x, y, cz*16 + z}, b, nil)
				}
			}
		}
	}
	<-w.Do(func(tx *world.Tx) {
		// (0,0): a small island.
		for x := -1; x <= 1; x++ {
			for z := -1; z <= 1; z++ {
				for y := 62; y <= 64; y++ {
					tx.SetBlock(cube.Pos{x, y, z}, block.Dirt{}, nil)
				}
			}
		}
		// (2,0) (3,0): the same stone cube, sub chunk 4 (y 64..79): other xz.
		fill(tx, 2, 0, 64, block.Stone{})
		fill(tx, 3, 0, 64, block.Stone{})
		// (4,0): the same cube one sub chunk higher (y 80..95).
		fill(tx, 4, 0, 80, block.Stone{})
		// (5,0): the same stone cube, but diamond went into the palette first.
		fill(tx, 5, 0, 64, block.Diamond{})
		fill(tx, 5, 0, 64, block.Stone{})
		// (6,0): stone with one dirt block: different content.
		fill(tx, 6, 0, 64, block.Stone{})
		tx.SetBlock(cube.Pos{6 * 16, 70, 0}, block.Dirt{}, nil)
	}).Done()
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.ID)
}
