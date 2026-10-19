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
	"strconv"
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
//	expect:<text>                wait for a chat message, command output, title, toast or sidebar line containing text
//	                             (a recipe list counts as "recipes=<n>")
//	expectre:<regexp>            wait for a chat message or command output matching regexp
//	show:<regexp>                like expectre, and print the message (e.g. a load line)
//	capture:<title>=<file>       wait for a form, write the invite code in it to file, close it
//	waitfile:<file>              wait until file exists
//	wait:<seconds>               sleep
//	chest:<title>                wait for an entity chest window whose title contains title
//	take:<slot>=ok|reject        take a stack of the open chest to the cursor, expect the response
//	close                        close the open chest
//	slot:<slot>=empty            wait for the server to empty an own inventory slot
//	touch:<entity type>          wait for an entity of the type (e.g. minecraft:allay) and use (right-click) it
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
			return ok && strings.Contains(formMarkers.ReplaceAllString(string(f.FormData), ""), title)
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
			return ok && strings.Contains(formMarkers.ReplaceAllString(string(f.FormData), ""), title)
		}).(*packet.ModalFormRequest)
		var data map[string]any
		_ = json.Unmarshal(pk.FormData, &data)
		fmt.Printf("FORM %q", data["title"])
		// Menus list their buttons in "buttons", or in "elements" next to
		// headers and labels (only buttons count for the answer).
		b, _ := data["buttons"].([]any)
		if b == nil {
			b, _ = data["elements"].([]any)
		}
		i := 0
		for _, e := range b {
			if e, ok := e.(map[string]any); ok && (e["type"] == nil || e["type"] == "button") {
				fmt.Printf(" [%d]%q", i, e["text"])
				i++
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
	case "chest":
		openChest(arg)
	case "take":
		slotStr, want, _ := strings.Cut(arg, "=")
		take(slotStr, want)
	case "touch":
		pk := waitFor(func(pk packet.Packet) bool {
			a, ok := pk.(*packet.AddActor)
			return ok && a.EntityType == arg
		}).(*packet.AddActor)
		_ = conn.WritePacket(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{
			TargetEntityRuntimeID: pk.EntityRuntimeID,
			ActionType:            protocol.UseItemOnEntityActionInteract,
			Position:              pk.Position,
		}})
		fmt.Println("> touched", arg)
	case "close":
		_ = conn.WritePacket(&packet.ContainerClose{WindowID: chestWindow})
		fmt.Println("> closed chest")
	case "slot":
		slotStr, _, _ := strings.Cut(arg, "=")
		slot, _ := strconv.Atoi(slotStr)
		waitFor(func(pk packet.Packet) bool {
			is, ok := pk.(*packet.InventorySlot)
			return ok && is.WindowID == protocol.WindowIDInventory && int(is.Slot) == slot && is.NewItem.Stack.Count == 0
		})
		fmt.Println("OK  slot emptied:", slot)
	case "show":
		re := regexp.MustCompile(arg)
		var shown string
		waitFor(func(pk packet.Packet) bool {
			holdForm(pk)
			for _, m := range messages(pk) {
				if re.MatchString(m) {
					shown = m
					return true
				}
			}
			return false
		})
		fmt.Println("OK  shown:", shown)
	case "expectre":
		re := regexp.MustCompile(arg)
		waitFor(func(pk packet.Packet) bool {
			holdForm(pk)
			for _, m := range messages(pk) {
				if re.MatchString(m) {
					return true
				}
			}
			return false
		})
		fmt.Println("OK  matched:", arg)
	case "expect":
		waitFor(func(pk packet.Packet) bool {
			holdForm(pk)
			for _, m := range messages(pk) {
				if strings.Contains(m, arg) {
					return true
				}
			}
			return false
		})
		fmt.Println("OK  saw:", arg)
	default:
		fail("unknown action %q", action)
	}
}

// formMarkers are the pack's form title markers ("§d§f<x>§r": no animation,
// list split, tree, create, back, ...). They come before the title, so a
// scenario matching `"title` from its first letter skips them.
var formMarkers = regexp.MustCompile(`§d§f§.§r`)

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

// messages returns the chat or command output text of a packet.
// itemNames are the names of the items of the join by network ID.
var itemNames = map[int32]string{}
var creativeIDs = map[int32]bool{}
var creativeRIDs = map[int32]map[int32]bool{}

func messages(pk packet.Packet) []string {
	switch pk := pk.(type) {
	case *packet.Text:
		return []string{pk.Message}
	case *packet.CommandOutput:
		var out []string
		for _, m := range pk.OutputMessages {
			out = append(out, m.Message)
		}
		return out
	case *packet.SetTitle:
		if pk.ActionType == packet.TitleActionSetTitle || pk.ActionType == packet.TitleActionTitleTextObject {
			return []string{pk.Text}
		}
	case *packet.ToastRequest:
		return []string{pk.Title + " " + pk.Message}
	case *packet.CraftingData:
		// The recipe list (sent again when recipes are locked or opened).
		n := len(pk.ShapedRecipes) + len(pk.ShapelessRecipes) + len(pk.MultiRecipes) + len(pk.UserDataShapelessRecipes) +
			len(pk.SmithingTransformRecipes) + len(pk.SmithingTrimRecipes) + len(pk.PotionRecipes) + len(pk.PotionContainerChangeRecipes)
		return []string{fmt.Sprintf("recipes=%d", n)}
	case *packet.SetDisplayObjective:
		// Sidebar title and lines count as messages too.
		return []string{pk.DisplayName}
	case *packet.SetScore:
		var out []string
		for _, e := range pk.Entries {
			out = append(out, e.DisplayName)
		}
		return out
	}
	return nil
}

// heldForms are forms that arrived while an expect step waited for a
// message: a toast may come after the form that follows it (the server
// holds quick toasts back), so the next form step still gets the form.
var heldForms []packet.Packet

// holdForm keeps a form passed by an expect step.
func holdForm(pk packet.Packet) {
	if _, ok := pk.(*packet.ModalFormRequest); ok {
		heldForms = append(heldForms, pk)
	}
}

func waitFor(match func(packet.Packet) bool) packet.Packet {
	for i, pk := range heldForms {
		if match(pk) {
			heldForms = append(heldForms[:i:i], heldForms[i+1:]...)
			return pk
		}
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			fail("waiting: %v", err)
		}
		clientReact(pk)
		if os.Getenv("BOT_TRACE") != "" {
			fmt.Printf("PKT %T\n", pk)
			if f, ok := pk.(*packet.ModalFormRequest); ok {
				fmt.Printf("FORMDATA %.300s\n", f.FormData)
			}
		}
		if _, ok := pk.(*packet.CommandOutput); ok {
			for _, m := range messages(pk) {
				fmt.Println("OUTPUT", m)
			}
		} else if t, ok := pk.(*packet.Text); ok {
			fmt.Println("CHAT", t.Message)
		} else if r, ok := pk.(*packet.ItemRegistry); ok {
			for _, e := range r.Items {
				itemNames[int32(e.RuntimeID)] = e.Name
			}
		} else if c, ok := pk.(*packet.CreativeContent); ok {
			for _, it := range c.Items {
				creativeIDs[it.Item.NetworkID] = true
				if creativeRIDs[it.Item.NetworkID] == nil {
					creativeRIDs[it.Item.NetworkID] = map[int32]bool{}
				}
				creativeRIDs[it.Item.NetworkID][it.Item.BlockRuntimeID] = true
				if os.Getenv("BOT_DUMP_RECIPES") != "" && (it.Item.NetworkID == 54 || it.Item.NetworkID == -213 || it.Item.NetworkID == 58) {
					fmt.Printf("CREATIVEDUMP id=%d meta=%d blockrid=%d\n", it.Item.NetworkID, it.Item.MetadataValue, it.Item.BlockRuntimeID)
				}
			}
			// The game's own items in the creative inventory (the recipe
			// book lists only recipes whose result is in it).
			for _, it := range c.Items {
				if name := itemNames[it.Item.NetworkID]; strings.HasPrefix(name, "dragonfactory:") || strings.HasSuffix(name, "chest") || strings.HasSuffix(name, "composter") {
					fmt.Println("CREATIVE", name)
				}
			}
		} else if c, ok := pk.(*packet.CraftingData); ok {
			// The results of the game's own recipes, for scenarios that
			// grep the log ("RECIPE dragonfactory:hammer").
			outputs := func(out []protocol.ItemStack) {
				for _, o := range out {
					if name := itemNames[o.NetworkID]; strings.HasPrefix(name, "dragonfactory:") || strings.HasSuffix(name, "chest") || strings.HasSuffix(name, "composter") {
						fmt.Println("RECIPE", name)
					}
				}
			}
			for _, r := range c.ShapelessRecipes {
				outputs(r.Output)
			}
			if os.Getenv("BOT_DUMP_RECIPES") != "" {
				mism := map[string]bool{}
				for _, o := range func() (all []protocol.ItemStack) {
					for _, r := range c.ShapedRecipes {
						all = append(all, r.Output...)
					}
					for _, r := range c.ShapelessRecipes {
						all = append(all, r.Output...)
					}
					return
				}() {
					if rids := creativeRIDs[o.NetworkID]; rids != nil && !rids[o.BlockRuntimeID] {
						mism[fmt.Sprintf("id=%d recipe-rid=%d creative=%v", o.NetworkID, o.BlockRuntimeID, rids)] = true
					}
				}
				for m := range mism {
					fmt.Println("MISMATCH", m)
				}
				for _, r := range c.ShapedRecipes {
					for _, o := range r.Output {
						fmt.Printf("RECIPEID shaped %d name=%q creative=%v meta=%d blockrid=%d\n", o.NetworkID, itemNames[o.NetworkID], creativeIDs[o.NetworkID], o.MetadataValue, o.BlockRuntimeID)
					}
				}
				for _, r := range c.ShapelessRecipes {
					for _, o := range r.Output {
						fmt.Printf("RECIPEID shapeless %d name=%q creative=%v\n", o.NetworkID, itemNames[o.NetworkID], creativeIDs[o.NetworkID])
					}
				}
			}
			for _, r := range c.ShapedRecipes {
				outputs(r.Output)
				if os.Getenv("BOT_DUMP_RECIPES") != "" && len(r.Output) > 0 && r.Output[0].NetworkID == 54 || r.Output[0].NetworkID == -213 || strings.HasPrefix(itemNames[r.Output[0].NetworkID], "dragonfactory:drawer") {
					fmt.Printf("SHAPEDDUMP prio=%d sym=%v block=%s\n", r.Priority, r.AssumeSymmetry, r.Block)
					for _, in := range r.Input {
						fmt.Printf("  IN count=%d %#v\n", in.Count, in.Descriptor)
					}
				}
			}
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

var (
	chestWindow byte
	chestSlots  []protocol.ItemInstance
	requestID   int32 = -1
)

// openChest waits for an entity-backed container: the invisible actor, the
// ContainerOpen pointing at it and the contents.
func openChest(title string) {
	var actor int64
	var size int32
	waitFor(func(pk packet.Packet) bool {
		a, ok := pk.(*packet.AddActor)
		if !ok || a.EntityType != "dragonfly:entity_container" {
			return false
		}
		name, _ := a.EntityMetadata[protocol.EntityDataKeyName].(string)
		size, _ = a.EntityMetadata[protocol.EntityDataKeyContainerSize].(int32)
		actor = a.EntityUniqueID
		return strings.Contains(name, title)
	})
	open := waitFor(func(pk packet.Packet) bool {
		o, ok := pk.(*packet.ContainerOpen)
		return ok && o.ContainerEntityUniqueID == actor
	}).(*packet.ContainerOpen)
	chestWindow = open.WindowID
	content := waitFor(func(pk packet.Packet) bool {
		c, ok := pk.(*packet.InventoryContent)
		return ok && c.WindowID == uint32(open.WindowID)
	}).(*packet.InventoryContent)
	chestSlots = content.Content
	filled := 0
	for _, it := range chestSlots {
		if it.Stack.Count > 0 {
			filled++
		}
	}
	fmt.Printf("CHEST %q window=%d size=%d slots=%d filled=%d\n", title, open.WindowID, size, len(chestSlots), filled)
	if int(size) != len(chestSlots) {
		fail("size %d but %d slots sent", size, len(chestSlots))
	}
}

// take moves a whole stack of the open chest to the cursor.
func take(slotStr, want string) {
	slot, _ := strconv.Atoi(slotStr)
	if slot >= len(chestSlots) {
		fail("no chest slot %d", slot)
	}
	it := chestSlots[slot]
	id := requestID
	requestID -= 2
	a := &protocol.TakeStackRequestAction{}
	a.Count = byte(it.Stack.Count)
	a.Source = protocol.StackRequestSlotInfo{Container: protocol.FullContainerName{ContainerID: protocol.ContainerLevelEntity}, Slot: byte(slot), StackNetworkID: it.StackNetworkID}
	a.Destination = protocol.StackRequestSlotInfo{Container: protocol.FullContainerName{ContainerID: protocol.ContainerCursor}}
	_ = conn.WritePacket(&packet.ItemStackRequest{Requests: []protocol.ItemStackRequest{{RequestID: id, Actions: []protocol.StackRequestAction{a}}}})
	resp := waitFor(func(pk packet.Packet) bool {
		r, ok := pk.(*packet.ItemStackResponse)
		return ok && len(r.Responses) > 0 && r.Responses[0].RequestID == id
	}).(*packet.ItemStackResponse)
	ok := resp.Responses[0].Status == protocol.ItemStackResponseStatusOK
	fmt.Printf("TAKE slot %d (count %d): ok=%v\n", slot, it.Stack.Count, ok)
	if ok != (want == "ok") {
		fail("take slot %d: ok=%v, want %s", slot, ok, want)
	}
}

// openWindows are the container windows the bot has open, like a client.
var openWindows = map[byte]bool{}

// clientReact answers packets the way the Bedrock client does, as far as
// the server relies on it: NetworkStackLatency requests, and a
// PacketViolationWarning for a ContainerOpen of a window already open
// (entity containers use it as the open acknowledgement).
func clientReact(pk packet.Packet) {
	switch pk := pk.(type) {
	case *packet.NetworkStackLatency:
		if pk.NeedsResponse {
			_ = conn.WritePacket(&packet.NetworkStackLatency{Timestamp: pk.Timestamp})
		}
	case *packet.ContainerOpen:
		if openWindows[pk.WindowID] {
			_ = conn.WritePacket(&packet.PacketViolationWarning{Type: -1, Severity: packet.ViolationSeverityWarning, PacketID: int32(pk.ID())})
		}
		openWindows[pk.WindowID] = true
	case *packet.ContainerClose:
		// A close that is not server side only acknowledges; the server's
		// open retries pair it with a new open.
		if pk.ServerSide {
			delete(openWindows, pk.WindowID)
		}
	}
}
