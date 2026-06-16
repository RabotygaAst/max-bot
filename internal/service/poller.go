package service

import (
	"context"
	"log/slog"
	"time"
)

type PollerConfig struct {
	Enabled        bool
	Limit          int
	TimeoutSeconds int
	RetryDelay     time.Duration
	Types          []string
}

type Poller struct {
	log     *slog.Logger
	service *BotService
	cfg     PollerConfig
	marker  *int64
}

func NewPoller(log *slog.Logger, service *BotService, cfg PollerConfig) *Poller {
	return &Poller{log: log, service: service, cfg: cfg}
}

func (p *Poller) Run(ctx context.Context) {
	if !p.cfg.Enabled {
		return
	}
	if p.cfg.Limit <= 0 {
		p.cfg.Limit = 100
	}
	if p.cfg.TimeoutSeconds < 0 {
		p.cfg.TimeoutSeconds = 30
	}
	if p.cfg.RetryDelay <= 0 {
		p.cfg.RetryDelay = 5 * time.Second
	}

	p.log.Info("max long polling started", "limit", p.cfg.Limit, "timeout_seconds", p.cfg.TimeoutSeconds, "types", p.cfg.Types)
	for ctx.Err() == nil {
		resp, err := p.service.max.GetUpdates(ctx, p.marker, p.cfg.Limit, p.cfg.TimeoutSeconds, p.cfg.Types)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			p.log.Error("max long polling request failed", "err", err, "retry_delay", p.cfg.RetryDelay)
			select {
			case <-time.After(p.cfg.RetryDelay):
			case <-ctx.Done():
			}
			continue
		}

		if resp.Marker != nil {
			p.marker = resp.Marker
		}
		for _, upd := range resp.Updates {
			p.service.ProcessUpdate(ctx, upd)
		}
	}
	p.log.Info("max long polling stopped")
}
