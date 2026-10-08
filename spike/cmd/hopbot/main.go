// Command hopbot connects to hopserver and hops between worlds twice.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func main() {
	d := minecraft.Dialer{
		IdentityData: login.IdentityData{DisplayName: "Hopper"},
		DownloadResourcePack: func(id uuid.UUID, version string, _, _ int) bool {
			fmt.Println("resource pack offered:", id, version)
			return true
		},
	}
	conn, err := d.Dial("raknet", "127.0.0.1:19132")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.DoSpawn(); err != nil {
		panic(err)
	}
	for _, p := range conn.ResourcePacks() {
		fmt.Println("resource pack downloaded:", p.Name(), "size", p.Len())
	}
	go func() {
		for range 3 {
			time.Sleep(time.Second)
			_ = conn.WritePacket(&packet.CommandRequest{CommandLine: "/hop", CommandOrigin: protocol.CommandOrigin{Origin: protocol.CommandOriginPlayer}})
		}
	}()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	chunks := 0
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			fmt.Println("chunks received:", chunks, "| end:", err)
			return
		}
		switch p := pk.(type) {
		case *packet.Text:
			fmt.Println("TEXT:", p.Message)
		case *packet.MovePlayer:
			fmt.Println("MOVE:", p.Position)
		case *packet.ChangeDimension:
			fmt.Println("DIMENSION CHANGE:", p.Dimension)
		case *packet.LevelChunk:
			chunks++
		case *packet.InventoryContent:
			n := 0
			for _, it := range p.Content {
				if it.Stack.NetworkID != 0 {
					n++
				}
			}
			if n > 0 {
				fmt.Println("INVENTORY window", p.WindowID, "non-empty slots:", n)
			}
		case *packet.AvailableActorIdentifiers:
			fmt.Println("ACTOR IDS contain item_holder:", strings.Contains(string(p.SerialisedEntityIdentifiers), "dragonfactory:item_holder"))
		case *packet.AddActor:
			fmt.Println("ADD ACTOR:", p.EntityType, "runtime", p.EntityRuntimeID, "pos", p.Position, "yaw", p.Yaw)
		case *packet.MobEquipment:
			if p.NewItem.Stack.NetworkID != 0 {
				fmt.Println("EQUIPMENT runtime", p.EntityRuntimeID, "window", p.WindowID, "network item id", p.NewItem.Stack.NetworkID)
			}
		case *packet.Disconnect:
			fmt.Println("DISCONNECTED:", p.Message)
		}
	}
}
