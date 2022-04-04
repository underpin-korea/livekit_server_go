package service

import (
	"context"
	"errors"

	"github.com/underpin-korea/protocol/livekit"
	"github.com/underpin-korea/protocol/logger"
	"github.com/underpin-korea/protocol/recording"
	"github.com/underpin-korea/protocol/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/underpin-korea/livekit_server_go/pkg/telemetry"
)

type RecordingService struct {
	bus       utils.MessageBus
	telemetry telemetry.TelemetryService
	shutdown  chan struct{}
}

func NewRecordingService(mb utils.MessageBus, telemetry telemetry.TelemetryService) *RecordingService {
	return &RecordingService{
		bus:       mb,
		telemetry: telemetry,
		shutdown:  make(chan struct{}, 1),
	}
}

func (s *RecordingService) Start() {
	if s.bus != nil {
		go s.resultsWorker()
	}
}

func (s *RecordingService) Stop() {
	s.shutdown <- struct{}{}
}

// RecordingService is deprecated, use EgressService instead
func (s *RecordingService) StartRecording(ctx context.Context, req *livekit.StartRecordingRequest) (*livekit.StartRecordingResponse, error) {
	if err := EnsureRecordPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.bus == nil {
		return nil, errors.New("recording not configured (redis required)")
	}

	// reserve a recorder
	recordingID, err := recording.ReserveRecorder(s.bus)
	if err != nil {
		return nil, err
	}

	// start the recording
	err = recording.RPC(ctx, s.bus, recordingID, &livekit.RecordingRequest{
		Request: &livekit.RecordingRequest_Start{
			Start: req,
		},
	})
	if err != nil {
		return nil, err
	}

	ri := &livekit.RecordingInfo{
		Id:     recordingID,
		Active: true,
	}

	switch template := req.Input.(type) {
	case *livekit.StartRecordingRequest_Template:
		ri.RoomName = template.Template.RoomName
	}

	logger.Debugw("recording started", "recordingID", recordingID)
	s.telemetry.RecordingStarted(ctx, ri)

	return &livekit.StartRecordingResponse{RecordingId: recordingID}, nil
}

// RecordingService is deprecated, use EgressService instead
func (s *RecordingService) AddOutput(ctx context.Context, req *livekit.AddOutputRequest) (*emptypb.Empty, error) {
	if err := EnsureRecordPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.bus == nil {
		return nil, errors.New("recording not configured (redis required)")
	}

	err := recording.RPC(ctx, s.bus, req.RecordingId, &livekit.RecordingRequest{
		Request: &livekit.RecordingRequest_AddOutput{
			AddOutput: req,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// RecordingService is deprecated, use EgressService instead
func (s *RecordingService) RemoveOutput(ctx context.Context, req *livekit.RemoveOutputRequest) (*emptypb.Empty, error) {
	if err := EnsureRecordPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.bus == nil {
		return nil, errors.New("recording not configured (redis required)")
	}

	err := recording.RPC(ctx, s.bus, req.RecordingId, &livekit.RecordingRequest{
		Request: &livekit.RecordingRequest_RemoveOutput{
			RemoveOutput: req,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// RecordingService is deprecated, use EgressService instead
func (s *RecordingService) EndRecording(ctx context.Context, req *livekit.EndRecordingRequest) (*emptypb.Empty, error) {
	if err := EnsureRecordPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.bus == nil {
		return nil, errors.New("recording not configured (redis required)")
	}

	err := recording.RPC(ctx, s.bus, req.RecordingId, &livekit.RecordingRequest{
		Request: &livekit.RecordingRequest_End{
			End: req,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *RecordingService) resultsWorker() {
	sub, err := s.bus.SubscribeQueue(context.Background(), recording.ResultChannel)
	if err != nil {
		logger.Errorw("failed to subscribe to results channel", err)
		return
	}

	resChan := sub.Channel()
	for {
		select {
		case msg := <-resChan:
			b := sub.Payload(msg)

			res := &livekit.RecordingInfo{}
			if err = proto.Unmarshal(b, res); err != nil {
				logger.Errorw("failed to read results", err)
				continue
			}

			// log results
			values := []interface{}{"recordingID", res.Id}
			if res.Error != "" {
				values = append(values, "error", res.Error)
			}
			logger.Debugw("recording ended", values...)

			s.telemetry.RecordingEnded(context.Background(), res)
		case <-s.shutdown:
			_ = sub.Close()
			return
		}
	}
}
