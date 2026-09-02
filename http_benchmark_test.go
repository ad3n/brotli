package brotli

import (
	"net/http"
	"os"
	"strconv"
	"testing"
)

type benchmarkResponseWriter struct {
	header http.Header
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (*benchmarkResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (*benchmarkResponseWriter) WriteHeader(int) {}

func BenchmarkHTTPCompressorBrotli(b *testing.B) {
	corpus, err := os.ReadFile("testdata/Isaac.Newton-Opticks.txt")
	if err != nil {
		b.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	request.Header.Set("Accept-Encoding", "br")

	for _, size := range []int{4 << 10, 64 << 10, 512 << 10} {
		payload := corpus[:size]
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			response := benchmarkResponseWriter{header: make(http.Header)}
			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			for i := 0; i < b.N; i++ {
				compressor := HTTPCompressor(&response, request)
				if _, err := compressor.Write(payload); err != nil {
					b.Fatal(err)
				}
				if err := compressor.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkNegotiateContentEncoding(b *testing.B) {
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	request.Header.Set("Accept-Encoding", "gzip;q=0.8, br;q=1.0, identity;q=0.1")
	offers := []string{"br", "gzip"}

	b.ReportAllocs()
	for b.Loop() {
		if encoding := negotiateContentEncoding(request, offers); encoding != "br" {
			b.Fatalf("encoding = %q, want br", encoding)
		}
	}
}
