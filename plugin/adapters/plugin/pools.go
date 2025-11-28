package plugin

import (
	"sync"

	pb "github.com/secmc/plugin/proto/generated/go"
)

// Message pools to reduce GC pressure
var (
	// envelopePool recycles EventEnvelope structs.
	// Consumers must call proto.Reset(e) before putting it back.
	envelopePool = sync.Pool{
		New: func() any {
			return &pb.EventEnvelope{}
		},
	}

	// playerMovePool recycles PlayerMoveEvent structs.
	playerMovePool = sync.Pool{
		New: func() any {
			return &pb.PlayerMoveEvent{}
		},
	}
)
