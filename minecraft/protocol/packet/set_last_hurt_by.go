package packet

import (
	"bytes"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// SetLastHurtBy is sent by the server to let the client know what entity type it was last hurt by. At this
// moment, the packet is useless and should not be used. There is no behaviour that depends on if this
// packet is sent or not.
type SetLastHurtBy struct {
	// EntityType is the numerical type of the entity that the player was last hurt by.
	EntityType int32
}

// ID ...
func (*SetLastHurtBy) ID() uint32 {
	return IDSetLastHurtBy
}

// Marshal ...
func (pk *SetLastHurtBy) Marshal(buf *bytes.Buffer) {
	_ = protocol.WriteVarint32(buf, pk.EntityType)
}

// Unmarshal ...
func (pk *SetLastHurtBy) Unmarshal(buf *bytes.Buffer) error {
	return protocol.Varint32(buf, &pk.EntityType)
}
