package session

// This file is part of the dragonfactory patch to Dragonfly (third_party/README.md):
// hiding HUD elements, locking client inputs and forgetting hidden entities.

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// HUD elements of SetHud (packet.HudElement*), all of them.
var hudElements = []int32{
	packet.HudElementPaperDoll, packet.HudElementArmour, packet.HudElementToolTips,
	packet.HudElementTouchControls, packet.HudElementCrosshair, packet.HudElementHotBar,
	packet.HudElementHealth, packet.HudElementProgressBar, packet.HudElementHunger,
	packet.HudElementAirBubbles, packet.HudElementHorseHealth, packet.HudElementStatusEffects,
	packet.HudElementItemText,
}

// HideHUD hides every vanilla HUD element of the client (hotbar, health,
// crosshair, ...), or shows them all again when hide is false. Resource pack
// HUD parts (the sidebar) are not vanilla elements and stay.
func (s *Session) HideHUD(hide bool) {
	if s == Nop {
		return
	}
	v := int32(packet.HudVisibilityReset)
	if hide {
		v = packet.HudVisibilityHide
	}
	s.writePacket(&packet.SetHud{Elements: hudElements, Visibility: v})
}

// LockInputs locks the client inputs in locks (packet.ClientInputLock*
// flags); 0 unlocks them all.
func (s *Session) LockInputs(locks uint32) {
	if s == Nop {
		return
	}
	s.writePacket(&packet.UpdateClientInputLocks{Locks: locks})
}

// ForgetHiddenEntities makes every entity hidden with StopShowingEntity
// visible again from then on, without spawning them now: the player is about
// to leave the world where they were hidden (the new world shows its
// entities as its chunks arrive).
func (s *Session) ForgetHiddenEntities() {
	s.entityMutex.Lock()
	clear(s.hiddenEntities)
	s.entityMutex.Unlock()
}
