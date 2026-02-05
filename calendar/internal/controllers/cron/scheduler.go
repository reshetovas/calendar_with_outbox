package cron

import (
	"context"
	"github.com/robfig/cron/v3"
	"time"
)

type Job interface {
	Run(ctx context.Context)
}

type Scheduler struct {
	c   *cron.Cron
	ctx context.Context
}

func NewScheduler(ctx context.Context) *Scheduler {
	// Поддерживаем стандартный cron формат и интервалы (@every)
	// Используем стандартный парсер, который поддерживает @every, @yearly, @monthly, @weekly, @daily, @hourly
	c := cron.New(
		cron.WithParser(cron.NewParser(
			cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		)),
	)
	return &Scheduler{c: c, ctx: ctx}
}

func (s *Scheduler) Add(spec string, job Job) (cron.EntryID, error) {
	return s.c.AddFunc(spec, func() {
		ctx, cancel := context.WithTimeout(s.ctx, 55*time.Minute)
		defer cancel()
		job.Run(ctx)
	})
}

func (s *Scheduler) Start() {
	s.c.Start()
}

func (s *Scheduler) Stop() {
	ctx := s.c.Stop()
	<-ctx.Done()
}
