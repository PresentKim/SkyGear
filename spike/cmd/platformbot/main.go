// Command platformbot is an end-to-end smoke test of the platform flow: it
// joins the server, answers the guide and save forms, creates/enters a save,
// returns to the lobby and enters again. Run it against `go run .` with
// authentication disabled.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var conn *minecraft.Conn

func main() {
	name := flag.String("name", "Bot", "player name")
	flag.Parse()
	var err error
	// A stable identity per name, so reconnecting is the same account.
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(*name)).String()
	conn, err = minecraft.Dialer{IdentityData: login.IdentityData{DisplayName: *name, Identity: id}}.Dial("raknet", "127.0.0.1:19132")
	if err != nil {
		fail("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.DoSpawn(); err != nil {
		fail("spawn: %v", err)
	}
	script := flag.Args()
	fmt.Println("script:", script)
	for _, action := range script {
		run(action)
	}
	fmt.Println("DONE")
	// Like a real client, say goodbye so the server ends the session now.
	_ = conn.WritePacket(&packet.Disconnect{})
	_ = conn.Flush()
	time.Sleep(200 * time.Millisecond)
}

// run executes one script action:
//
//	form:<title>=<button index>   wait for a form with title and press a button
//	input:<title>=<text>         wait for a custom form and submit text
//	cmd:/text                    run a command
//	expect:<text>                wait for a chat message containing text
//	capture:<title>=<file>       wait for a form, write the invite code in it to file, close it
//	waitfile:<file>              wait until file exists
//	wait:<seconds>               sleep
//
// "@file" in cmd and input values is replaced by the file's contents.
func run(action string) {
	kind, arg, _ := strings.Cut(action, ":")
	switch kind {
	case "wait":
		d, _ := time.ParseDuration(arg + "s")
		time.Sleep(d)
	case "waitfile":
		for i := 0; i < 300; i++ {
			if _, err := os.Stat(arg); err == nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		fail("file %s never appeared", arg)
	case "capture":
		title, file, _ := strings.Cut(arg, "=")
		pk := waitFor(func(pk packet.Packet) bool {
			f, ok := pk.(*packet.ModalFormRequest)
			return ok && strings.Contains(string(f.FormData), title)
		}).(*packet.ModalFormRequest)
		code := regexp.MustCompile(`[A-Z2-9]{6}`).FindString(regexp.MustCompile(`§.`).ReplaceAllString(string(pk.FormData), ""))
		if code == "" {
			fail("no code in form %s", pk.FormData)
		}
		_ = os.WriteFile(file, []byte(code), 0o644)
		_ = conn.WritePacket(&packet.ModalFormResponse{FormID: pk.FormID})
		fmt.Println("CAPTURED", code)
	case "cmd":
		arg = expand(arg)
		_ = conn.WritePacket(&packet.CommandRequest{CommandLine: arg, CommandOrigin: protocol.CommandOrigin{Origin: protocol.CommandOriginPlayer}})
		fmt.Println("> ", arg)
	case "form", "input":
		title, val, _ := strings.Cut(arg, "=")
		pk := waitFor(func(pk packet.Packet) bool {
			f, ok := pk.(*packet.ModalFormRequest)
			return ok && strings.Contains(string(f.FormData), title)
		}).(*packet.ModalFormRequest)
		var data map[string]any
		_ = json.Unmarshal(pk.FormData, &data)
		fmt.Printf("FORM %q", data["title"])
		if b, ok := data["buttons"].([]any); ok {
			for i, btn := range b {
				fmt.Printf(" [%d]%q", i, btn.(map[string]any)["text"])
			}
		}
		fmt.Println()
		val = expand(val)
		resp := []byte(val)
		if kind == "input" {
			resp, _ = json.Marshal([]any{val})
		}
		_ = conn.WritePacket(&packet.ModalFormResponse{FormID: pk.FormID, ResponseData: protocol.Option(resp)})
		fmt.Println("> answered", string(resp))
	case "expect":
		waitFor(func(pk packet.Packet) bool {
			t, ok := pk.(*packet.Text)
			return ok && strings.Contains(t.Message, arg)
		})
		fmt.Println("OK  saw:", arg)
	default:
		fail("unknown action %q", action)
	}
}

// expand replaces "@file" with the file's contents.
func expand(s string) string {
	return regexp.MustCompile(`@(\S+)`).ReplaceAllStringFunc(s, func(m string) string {
		b, err := os.ReadFile(m[1:])
		if err != nil {
			fail("read %s: %v", m[1:], err)
		}
		return strings.TrimSpace(string(b))
	})
}

func waitFor(match func(packet.Packet) bool) packet.Packet {
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			fail("waiting: %v", err)
		}
		if t, ok := pk.(*packet.Text); ok {
			fmt.Println("CHAT", t.Message)
		}
		if match(pk) {
			return pk
		}
	}
}

func fail(f string, a ...any) {
	fmt.Printf("FAIL "+f+"\n", a...)
	os.Exit(1)
}
