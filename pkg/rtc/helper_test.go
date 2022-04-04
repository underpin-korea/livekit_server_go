package rtc_test

import (
	"github.com/underpin-korea/protocol/livekit"
	"github.com/underpin-korea/protocol/utils"

	"github.com/underpin-korea/livekit_server_go/pkg/rtc/types"
	"github.com/underpin-korea/livekit_server_go/pkg/rtc/types/typesfakes"
)

func newMockParticipant(identity livekit.ParticipantIdentity, protocol types.ProtocolVersion, hidden bool) *typesfakes.FakeLocalParticipant {
	p := &typesfakes.FakeLocalParticipant{}
	sid := utils.NewGuid(utils.ParticipantPrefix)
	p.IDReturns(livekit.ParticipantID(sid))
	p.IdentityReturns(identity)
	p.StateReturns(livekit.ParticipantInfo_JOINED)
	p.ProtocolVersionReturns(protocol)
	p.CanSubscribeReturns(true)
	p.CanPublishReturns(!hidden)
	p.CanPublishDataReturns(!hidden)
	p.HiddenReturns(hidden)
	p.ToProtoReturns(&livekit.ParticipantInfo{
		Sid:      sid,
		Identity: string(identity),
		State:    livekit.ParticipantInfo_JOINED,
	})

	p.SetMetadataStub = func(m string) {
		var f func(participant types.LocalParticipant)
		if p.OnParticipantUpdateCallCount() > 0 {
			f = p.OnParticipantUpdateArgsForCall(p.OnParticipantUpdateCallCount() - 1)
		}
		if f != nil {
			f(p)
		}
	}
	updateTrack := func() {
		var f func(participant types.LocalParticipant, track types.MediaTrack)
		if p.OnTrackUpdatedCallCount() > 0 {
			f = p.OnTrackUpdatedArgsForCall(p.OnTrackUpdatedCallCount() - 1)
		}
		if f != nil {
			f(p, nil)
		}
	}

	p.SetTrackMutedStub = func(sid livekit.TrackID, muted bool, fromServer bool) {
		updateTrack()
	}
	p.AddTrackStub = func(req *livekit.AddTrackRequest) {
		updateTrack()
	}

	return p
}

func newMockTrack(kind livekit.TrackType, name string) *typesfakes.FakeMediaTrack {
	t := &typesfakes.FakeMediaTrack{}
	t.IDReturns(livekit.TrackID(utils.NewGuid(utils.TrackPrefix)))
	t.KindReturns(kind)
	t.NameReturns(name)
	return t
}
