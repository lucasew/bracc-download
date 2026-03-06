package main

import (
	"fmt"
	"path"
	"sync"
	"time"

	"bracc/pkg/provider"

	"github.com/sethgrid/multibar"
)

type multibarFactory struct {
	container *multibar.BarContainer
	mu        sync.Mutex
}

func newMultibarFactory() (*multibarFactory, error) {
	container, err := multibar.New()
	if err != nil {
		return nil, err
	}
	return &multibarFactory{container: container}, nil
}

func (f *multibarFactory) NewBar(job provider.Job) provider.ProgressBar {
	f.mu.Lock()
	defer f.mu.Unlock()

	u := job.GetURL()
	label := path.Base(u.Path)
	if label == "." || label == "/" || label == "" {
		label = u.Host
	}

	_ = f.container.MakeBar(1, label)
	bar := f.container.Bars[len(f.container.Bars)-1]
	bar.ShowPercent = false

	return &multibarProgressBar{
		bar:          bar,
		lastRendered: time.Now(),
		current:      -1,
	}
}

type multibarProgressBar struct {
	mu           sync.Mutex
	bar          *multibar.ProgressBar
	total        int64
	current      int64
	lastRendered time.Time
}

func (b *multibarProgressBar) SetTotal(total int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if total <= 0 {
		return
	}
	b.total = total
	b.bar.Total = int(total)
	b.bar.ShowPercent = true
	b.renderLocked(false)
}

func (b *multibarProgressBar) SetCurrent(current int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if current < 0 {
		current = 0
	}
	b.current = current
	if b.total == 0 && current > 0 {
		b.bar.Total = int(current)
	}
	b.renderLocked(false)
}

func (b *multibarProgressBar) Complete(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil && b.total > 0 {
		b.current = b.total
	}
	if err != nil {
		b.bar.AddPrepend(fmt.Sprintf("%s [failed]", b.bar.Prepend))
	}
	b.renderLocked(true)
}

func (b *multibarProgressBar) renderLocked(force bool) {
	now := time.Now()
	if !force && now.Sub(b.lastRendered) < 100*time.Millisecond {
		return
	}
	progress := b.current
	if progress < 0 {
		progress = 0
	}
	if b.total == 0 && progress == 0 {
		b.bar.Total = 1
	}
	if b.total == 0 && progress > int64(b.bar.Total) {
		b.bar.Total = int(progress)
	}
	b.bar.Update(int(progress))
	b.lastRendered = now
}
