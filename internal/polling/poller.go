package polling

import (
	"context"
	"log/slog"
	"time"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/service"
)

type Poller struct {
	log        *slog.Logger
	max        *maxclient.Client
	service    *service.BotService
	limit      int
	timeout    time.Duration
	retryDelay time.Duration
	types      []string
}

func New(log *slog.Logger, max *maxclient.Client, service *service.BotService, limit int, timeout, retryDelay time.Duration, types []string) *Poller {
	return &Poller{log: log, max: max, service: service, limit: limit, timeout: timeout, retryDelay: retryDelay, types: types}
}

func (p *Poller) Run(ctx context.Context) error {
	var marker *int64
	p.log.Info("max long polling started", "limit", p.limit, "timeout_seconds", int(p.timeout.Seconds()), "types", p.types)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updatesCtx, cancel := context.WithTimeout(ctx, p.timeout+p.retryDelay)
		resp, err := p.max.GetUpdates(updatesCtx, marker, p.limit, int(p.timeout.Seconds()), p.types)
		cancel()
		if err != nil {
			p.log.Error("get max updates failed", "err", err)
			if !sleep(ctx, p.retryDelay) {
				return ctx.Err()
			}
			continue
		}

		if resp.Marker != nil {
			marker = resp.Marker
		}
		if len(resp.Updates) == 0 {
			continue
		}

		p.log.Info("max updates received", "count", len(resp.Updates), "marker", markerValue(marker))
		for _, upd := range resp.Updates {
			if !processable(upd) {
				p.log.Info("max update skipped", "update_type", upd.UpdateType, "event_id", upd.EventID())
				continue
			}
			p.service.ProcessUpdate(ctx, upd)
		}
	}
}

func processable(upd model.MAXUpdate) bool {
	return upd.ChatID() != 0 && upd.UserID() != 0 && upd.Text() != ""
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func markerValue(marker *int64) any {
	if marker == nil {
		return nil
	}
	return *marker
}
