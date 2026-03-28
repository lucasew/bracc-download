package simple

import (
	"net/http"
	"net/url"
	"testing"
)

func TestFilenameFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		cdHeader string
		expected string
	}{
		{
			name:     "from URL path",
			url:      "https://example.com/data/report.csv",
			cdHeader: "",
			expected: "report.csv",
		},
		{
			name:     "from URL host when path is empty",
			url:      "https://example.com/",
			cdHeader: "",
			expected: "example.com",
		},
		{
			name:     "from Content-Disposition filename",
			url:      "https://example.com/download",
			cdHeader: `attachment; filename="data.zip"`,
			expected: "data.zip",
		},
		{
			name:     "from Content-Disposition filename* with path traversal",
			url:      "https://example.com/download",
			cdHeader: `attachment; filename*="UTF-8''../../../etc/passwd"`,
			expected: "passwd",
		},
		{
			name:     "from Content-Disposition filename with path traversal",
			url:      "https://example.com/download",
			cdHeader: `attachment; filename="../../../etc/passwd"`,
			expected: "passwd",
		},
		{
			name:     "from URL with path traversal",
			url:      "https://example.com/data/../../../etc/passwd",
			cdHeader: "",
			expected: "passwd",
		},
		{
			name:     "from invalid Content-Disposition falls back to URL",
			url:      "https://example.com/data/report.csv",
			cdHeader: "invalid header format",
			expected: "report.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqUrl, _ := url.Parse(tt.url)
			req := &http.Request{URL: reqUrl}
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}
			if tt.cdHeader != "" {
				resp.Header.Set("Content-Disposition", tt.cdHeader)
			}

			actual := filenameFromResponse(resp)
			if actual != tt.expected {
				t.Errorf("filenameFromResponse() = %q, want %q", actual, tt.expected)
			}
		})
	}
}
