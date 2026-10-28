package ui

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPlainText(t *testing.T) {
	name := i18n.ClientKey("item.diamond.name", "Diamond")
	form := &packet.ModalFormRequest{FormData: []byte(`{"title":"` + name + `"}`)}
	text := &packet.Text{Message: "got " + name, Parameters: []string{name}}
	for _, pk := range []packet.Packet{form, text} {
		if plainText(nil, pk) {
			t.Fatal("plainText dropped a packet")
		}
	}
	if string(form.FormData) != `{"title":"Diamond"}` || text.Message != "got Diamond" || text.Parameters[0] != "Diamond" {
		t.Fatalf("plainText left spans: %s / %q / %q", form.FormData, text.Message, text.Parameters)
	}
}
