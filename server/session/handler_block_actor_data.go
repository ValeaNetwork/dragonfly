package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"strings"
	"unicode/utf8"
)

// BlockActorDataHandler handles an incoming BlockActorData packet from the client, sent for some block entities like
// signs when they are edited.
type BlockActorDataHandler struct{}

// Handle ...
func (b BlockActorDataHandler) Handle(p packet.Packet, s *Session) error {
	pk := p.(*packet.BlockActorData)
	if id, ok := pk.NBTData["id"]; ok {
		pos := cube.Pos{int(pk.Position.X()), int(pk.Position.Y()), int(pk.Position.Z())}
		switch id {
		case "Sign":
			return b.handleSign(pk, pos, s)
		}
		return fmt.Errorf("unhandled block actor data ID %v", id)
	}
	return fmt.Errorf("block actor data without 'id' tag: %v", pk.NBTData)
}

// handleSign handles the BlockActorData packet sent when editing a sign.
func (b BlockActorDataHandler) handleSign(pk *packet.BlockActorData, pos cube.Pos, s *Session) error {
	if _, ok := s.c.World().Block(pos).(block.Sign); !ok {
		s.log.Debugf("sign block actor data for position without sign %v", pos)
		return nil
	}

	frontText, err := b.textFromNBTData(pk.NBTData, true)
	if err != nil {
		return err
	}
	backText, err := b.textFromNBTData(pk.NBTData, false)
	if err != nil {
		return err
	}
	if err := s.c.EditSign(pos, frontText, backText); err != nil {
		return err
	}
	return nil
}

// textFromNBTData attempts to retrieve the text from the NBT data of specific sign from the BlockActorData packet.
func (b BlockActorDataHandler) textFromNBTData(data map[string]any, frontSide bool) (string, error) {
	var sideData map[string]any
	var side string
	if frontSide {
		frontSide, ok := data["FrontText"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("sign block actor data 'FrontText' tag was not found or was not a map: %#v", data["FrontText"])
		}
		sideData = frontSide
		side = "front"
	} else {
		backSide, ok := data["BackText"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("sign block actor data 'BackText' tag was not found or was not a map: %#v", data["BackText"])
		}
		sideData = backSide
		side = "back"
	}
	var text string
	pkText, ok := sideData["Text"]
	if !ok {
		return "", fmt.Errorf("sign block actor data had no 'Text' tag for side %s", side)
	}
	if text, ok = pkText.(string); !ok {
		return "", fmt.Errorf("sign block actor data 'Text' tag was not a string for side %s: %#v", side, pkText)
	}

	// Verify that the text was valid. It must be valid UTF8 and not more than 100 characters long.
	text = strings.TrimRight(text, "\n")
	if len(text) > 256 {
		return "", fmt.Errorf("sign block actor data text was longer than 256 characters for side %s", side)
	}
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("sign block actor data text was not valid UTF8 for side %s", side)
	}
	return text, nil
}
