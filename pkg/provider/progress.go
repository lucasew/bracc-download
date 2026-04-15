package provider

import (
	"context"
	"io"
)

type ProgressBar interface {
	SetName(name string)
	SetTotal(total int64)
	SetCurrent(current int64)
	Complete(err error)
}

type ProgressFactory interface {
	NewBar(job Job) ProgressBar
}

type progressFactoryKey struct{}
type progressBarKey struct{}

type nopProgressBar struct{}

func (nopProgressBar) SetName(string)   {}
func (nopProgressBar) SetTotal(int64)   {}
func (nopProgressBar) SetCurrent(int64) {}
func (nopProgressBar) Complete(error)   {}

func WithProgressFactory(ctx context.Context, factory ProgressFactory) context.Context {
	return context.WithValue(ctx, progressFactoryKey{}, factory)
}

func WithProgressBar(ctx context.Context, bar ProgressBar) context.Context {
	return context.WithValue(ctx, progressBarKey{}, bar)
}

func progressBarFromContext(ctx context.Context) ProgressBar {
	bar, ok := ctx.Value(progressBarKey{}).(ProgressBar)
	if !ok || bar == nil {
		return nopProgressBar{}
	}
	return bar
}

func ProgressBarFromContext(ctx context.Context) ProgressBar {
	return progressBarFromContext(ctx)
}

func progressFactoryFromContext(ctx context.Context) ProgressFactory {
	factory, ok := ctx.Value(progressFactoryKey{}).(ProgressFactory)
	if !ok {
		return nil
	}
	return factory
}

func CopyWithProgress(ctx context.Context, job Job, dst io.Writer, src io.Reader, total int64) (int64, error) {
	bar := progressBarFromContext(ctx)
	if total > 0 {
		bar.SetTotal(total)
	}

	buf := make([]byte, 128*1024)
	var downloaded int64
	_ = job

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			downloaded += int64(nw)
			bar.SetCurrent(downloaded)
			if ew != nil {
				return downloaded, ew
			}
			if nw != nr {
				return downloaded, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return downloaded, er
		}
	}

	bar.SetCurrent(downloaded)
	return downloaded, nil
}
