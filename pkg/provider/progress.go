package provider

import (
	"context"
	"io"
)

// CopyWithProgress performs a chunked copy of io.Reader to io.Writer while periodically
// updating the ProgressBar embedded inside the provided context. If no progress bar
// exists in context, a no-op fallback acts silently. The 'total' parameter optionally
// hints at the expected file size, improving bar accuracy.
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
