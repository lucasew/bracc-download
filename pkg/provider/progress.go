package provider

import (
	"context"
	"io"
	"time"
)

func CopyWithProgress(ctx context.Context, job Job, dst io.Writer, src io.Reader, total int64) (int64, error) {
	buf := make([]byte, 128*1024)
	var downloaded int64
	lastReport := time.Time{}

	ReportProgress(ctx, job, JobProgress{
		State:           JobStateDownloading,
		DownloadedBytes: 0,
		TotalBytes:      total,
	})

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			downloaded += int64(nw)
			now := time.Now()
			if lastReport.IsZero() || now.Sub(lastReport) >= 200*time.Millisecond || (total > 0 && downloaded >= total) {
				ReportProgress(ctx, job, JobProgress{
					State:           JobStateDownloading,
					DownloadedBytes: downloaded,
					TotalBytes:      total,
				})
				lastReport = now
			}
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

	ReportProgress(ctx, job, JobProgress{
		State:           JobStateDownloading,
		DownloadedBytes: downloaded,
		TotalBytes:      total,
	})
	return downloaded, nil
}
