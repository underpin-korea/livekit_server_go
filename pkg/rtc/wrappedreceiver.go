package rtc

import (
	"github.com/underpin-korea/protocol/livekit"

	"github.com/underpin-korea/livekit_server_go/pkg/sfu"
)

// wrapper around WebRTC receiver, overriding its ID

type WrappedReceiver struct {
	sfu.TrackReceiver
	trackID  livekit.TrackID
	streamId string
}

func NewWrappedReceiver(receiver sfu.TrackReceiver, trackID livekit.TrackID, streamId string) WrappedReceiver {
	return WrappedReceiver{
		TrackReceiver: receiver,
		trackID:       trackID,
		streamId:      streamId,
	}
}

func (r WrappedReceiver) TrackID() livekit.TrackID {
	return r.trackID
}

func (r WrappedReceiver) StreamID() string {
	return r.streamId
}
