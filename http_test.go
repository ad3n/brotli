package brotli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNegotiateContentEncoding(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{header: "gzip;q=0.8, br;q=1.0", want: "br"},
		{header: "gzip;q=1.0, br;q=1.0", want: "br"},
		{header: "gzip;q=1.0, br;q=0.5", want: "gzip"},
		{header: "br;q=0, gzip;q=0", want: ""},
		{header: "identity", want: "identity"},
		{header: "one, two, three, four, five, six, seven, eight, br", want: "br"},
	}

	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", test.header)
		if got := negotiateContentEncoding(request, []string{"br", "gzip"}); got != test.want {
			t.Errorf("negotiateContentEncoding(%q) = %q, want %q", test.header, got, test.want)
		}
	}
}

func TestHTTPCompressorBrotliConcurrentReuse(t *testing.T) {
	const workers = 8
	const iterations = 20

	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				payload := bytes.Repeat(fmt.Appendf(nil, "worker=%d iteration=%d ", worker, iteration), 128)
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				request.Header.Set("Accept-Encoding", "br")
				response := httptest.NewRecorder()
				compressor := HTTPCompressor(response, request)
				if _, err := compressor.Write(payload); err != nil {
					errCh <- err

					return
				}
				if err := compressor.Close(); err != nil {
					errCh <- err

					return
				}

				decoded, err := io.ReadAll(NewReader(response.Body))
				if err != nil {
					errCh <- err

					return
				}
				if !bytes.Equal(decoded, payload) {
					errCh <- fmt.Errorf("worker %d iteration %d: decoded payload mismatch", worker, iteration)

					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestHTTPCompressorBrotliWriterLifecycle(t *testing.T) {
	payloads := [][]byte{
		bytes.Repeat([]byte("first request "), 256),
		bytes.Repeat([]byte("second request "), 256),
	}

	for _, payload := range payloads {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "br")
		response := httptest.NewRecorder()
		compressor := HTTPCompressor(response, request)
		if _, err := compressor.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := compressor.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := compressor.Write(payload); !errors.Is(err, errWriterClosed) {
			t.Fatalf("Write after Close error = %v, want %v", err, errWriterClosed)
		}
		if err := compressor.Close(); !errors.Is(err, errWriterClosed) {
			t.Fatalf("second Close error = %v, want %v", err, errWriterClosed)
		}

		decoded, err := io.ReadAll(NewReader(response.Body))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded, payload) {
			t.Fatalf("decoded payload length = %d, want %d", len(decoded), len(payload))
		}
	}
}
