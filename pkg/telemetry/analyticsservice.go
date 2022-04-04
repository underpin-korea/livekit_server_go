package telemetry

import (
	"context"

	"github.com/underpin-korea/protocol/livekit"
	"github.com/underpin-korea/protocol/logger"

	"github.com/underpin-korea/livekit_server_go/pkg/config"
	"github.com/underpin-korea/livekit_server_go/pkg/routing"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . AnalyticsService
type AnalyticsService interface {
	SendStats(ctx context.Context, stats []*livekit.AnalyticsStat)
	SendEvent(ctx context.Context, events *livekit.AnalyticsEvent)
}

type analyticsService struct {
	analyticsKey string
	nodeID       string

	events livekit.AnalyticsRecorderService_IngestEventsClient
	stats  livekit.AnalyticsRecorderService_IngestStatsClient
}

func NewAnalyticsService(_ *config.Config, currentNode routing.LocalNode) AnalyticsService {
	return &analyticsService{
		analyticsKey: "", // TODO: conf.AnalyticsKey
		nodeID:       currentNode.Id,
	}
}

func (a *analyticsService) SendStats(_ context.Context, stats []*livekit.AnalyticsStat) {
	if a.stats == nil {
		return
	}

	for _, stat := range stats {
		stat.AnalyticsKey = a.analyticsKey
		stat.Node = a.nodeID
	}
	if err := a.stats.Send(&livekit.AnalyticsStats{Stats: stats}); err != nil {
		logger.Errorw("failed to send stats", err)
	}
}

func (a *analyticsService) SendEvent(_ context.Context, event *livekit.AnalyticsEvent) {
	if a.events == nil {
		return
	}

	event.AnalyticsKey = a.analyticsKey
	if err := a.events.Send(&livekit.AnalyticsEvents{
		Events: []*livekit.AnalyticsEvent{event},
	}); err != nil {
		logger.Errorw("failed to send event", err, "eventType", event.Type.String())
	}
}
