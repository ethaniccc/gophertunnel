package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CorrectionTypePlayer = iota
	CorrectionTypeVechile
)

// CorrectPlayerMovePrediction is sent by the server if and only if StartGame.ServerAuthoritativeMovementMode
// is set to AuthoritativeMovementModeServerWithRewind. The packet is used to correct movement at a specific
// point in time.
type CorrectPlayerMovePrediction struct {
	// Position is the position that the player is supposed to be at the tick written in the field below.
	// The client will change its current position based on movement after that tick starting from the
	// Position.
	Position mgl32.Vec3
	// Delta is the change in position compared to what the client sent as its position at that specific tick.
	Delta mgl32.Vec3
	// OnGround specifies if the player was on the ground at the time of the tick below.
	OnGround bool
	// Tick is the tick of the movement which was corrected by this packet.
	Tick uint64
	// CorrectionType is the correction type send to the player. This influences how the rewind is performed
	// on the client.
	CorrectionType byte
}

// ID ...
func (*CorrectPlayerMovePrediction) ID() uint32 {
	return IDCorrectPlayerMovePrediction
}

func (pk *CorrectPlayerMovePrediction) Marshal(io protocol.IO) {
	io.Vec3(&pk.Position)
	io.Vec3(&pk.Delta)
	io.Bool(&pk.OnGround)
	io.Varuint64(&pk.Tick)
	io.Uint8(&pk.CorrectionType)
}
