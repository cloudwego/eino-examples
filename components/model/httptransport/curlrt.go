/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package httptransport provides a configurable cURL-style logging RoundTripper
// for HTTP-based ChatModel clients. It logs the real outbound HTTP request
// (as a cURL command) and the inbound HTTP response. Streaming responses (SSE
// or NDJSON) can be logged chunk-by-chunk without breaking the stream.
//
// Quick usage:
//
//	client := &http.Client{Transport: httptransport.NewCurlRT(
//	    http.DefaultTransport,
//	    httptransport.WithLogger(log.Default()),
//	    // Or pass a context-aware logger to extract request IDs:
//	    httptransport.WithCtxLogger(httptransport.IDCtxLogger{L: log.Default()}),
//	    // Security controls:
//	    httptransport.WithPrintAuth(false),                     // mask Authorization
//	    httptransport.WithMaskHeaders([]string{"X-API-KEY"}),  // mask custom headers
//	    // Streaming controls:
//	    httptransport.WithStreamLogging(true),
//	    httptransport.WithMaxStreamLogBytes(8192),
//	)}
//	cm, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{ HTTPClient: client, ... })
//
// Notes:
//   - WithCtxLogger is preferred when you carry a request/log ID in context.
//   - WithPrintAuth controls whether the Authorization header is printed.
//   - WithMaskHeaders and WithMaskFunc allow masking arbitrary headers.
//   - When stream logging is enabled, headers are logged once, and chunks are
//     emitted as they are read. With a plain Logger, a capped summary is printed
//     on Close(); with a CtxLogger, each chunk is logged directly.
package httptransport

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
)

// sanitizeLogValue removes line breaks and carriage returns to prevent log forging
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// Logger is a minimal printf-style logger used when no context is required.
type Logger interface{ Printf(string, ...any) }

// CtxLogger is a context-aware logger; use this to inject request IDs or
// structured logging derived from the HTTP request context.
type CtxLogger interface {
	Printf(context.Context, string, ...any)
}

// CurlRT is an http.RoundTripper that logs requests/responses in cURL style.
// Configure it with the CurlOption helpers via NewCurlRT.
type CurlRT struct {
	base              http.RoundTripper
	logger            Logger
	ctxLogger         CtxLogger
	printAuth         bool
	maskHeaders       map[string]struct{}
	maskFn            func(string, string) string
	streamEnabled     bool
	maxStreamLogBytes int
	streamCTFilter    func(string) bool
}

// CurlOption configures CurlRT behavior.
type CurlOption func(*CurlRT)

// WithLogger sets a simple printf-style logger.
func WithLogger(l Logger) CurlOption { return func(c *CurlRT) { c.logger = l } }

// WithCtxLogger sets a context-aware logger for request/response/chunk logs.
func WithCtxLogger(l CtxLogger) CurlOption { return func(c *CurlRT) { c.ctxLogger = l } }

// WithPrintAuth controls whether the Authorization header value is printed.
func WithPrintAuth(b bool) CurlOption { return func(c *CurlRT) { c.printAuth = b } }

// WithMaskHeaders masks specified header names (case-insensitive) in logs.
func WithMaskHeaders(names []string) CurlOption {
	return func(c *CurlRT) {
		if c.maskHeaders == nil {
			c.maskHeaders = make(map[string]struct{})
		}
		for _, n := range names {
			c.maskHeaders[strings.ToLower(n)] = struct{}{}
		}
	}
}

// WithMaskFunc provides a custom masking function for header values.
func WithMaskFunc(f func(name, value string) string) CurlOption {
	return func(c *CurlRT) { c.maskFn = f }
}

// WithStreamLogging enables logging for streaming responses (SSE/NDJSON).
func WithStreamLogging(enabled bool) CurlOption { return func(c *CurlRT) { c.streamEnabled = enabled } }

// WithMaxStreamLogBytes caps stream summary size when using a plain Logger.
func WithMaxStreamLogBytes(n int) CurlOption { return func(c *CurlRT) { c.maxStreamLogBytes = n } }

// WithStreamContentTypeFilter sets a filter to detect streaming responses.
func WithStreamContentTypeFilter(f func(ct string) bool) CurlOption {
	return func(c *CurlRT) { c.streamCTFilter = f }
}

func NewCurlRT(base http.RoundTripper, opts ...CurlOption) *CurlRT {
	rt := &CurlRT{base: base}
	for _, o := range opts {
		o(rt)
	}
	if rt.logger == nil {
		rt.logger = log.Default()
	}
	if rt.maskFn == nil {
		rt.maskFn = func(_ string, _ string) string { return "<redacted>" }
	}
	if rt.streamCTFilter == nil {
		rt.streamCTFilter = func(ct string) bool {
			ct = strings.ToLower(ct)
			return strings.Contains(ct, "text/event-stream") || strings.Contains(ct, "application/x-ndjson")
		}
	}
	return rt
}

func (c *CurlRT) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}
	curl := c.buildCurl(req, reqBody)
	if c.ctxLogger != nil {
		c.ctxLogger.Printf(req.Context(), "[curl request] %s", curl)
	} else {
		c.logger.Printf("[curl request] %s", curl)
	}

	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	ct := resp.Header.Get("Content-Type")
	if c.streamEnabled && c.streamCTFilter(ct) {
		if c.ctxLogger != nil {
			c.ctxLogger.Printf(req.Context(), "[curl response] HTTP/%d.%d %d\n%s\n\n(streaming...)", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, c.formatHeaders(resp.Header))
		} else {
			c.logger.Printf("[curl response] HTTP/%d.%d %d\n%s\n\n(streaming...)", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, c.formatHeaders(resp.Header))
		}
		resp.Body = newLoggingReadCloser(resp.Body, req.Context(), c)
		return resp, nil
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
	}
	if c.ctxLogger != nil {
		c.ctxLogger.Printf(req.Context(), "[curl response] HTTP/%d.%d %d\n%s\n\n%s", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, c.formatHeaders(resp.Header), string(respBody))
	} else {
		c.logger.Printf("[curl response] HTTP/%d.%d %d\n%s\n\n%s", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, c.formatHeaders(resp.Header), string(respBody))
	}
	return resp, nil
}

func (c *CurlRT) mask(name, value string) string {
	if strings.EqualFold(name, "Authorization") && !c.printAuth {
		return "<redacted>"
	}
	if _, ok := c.maskHeaders[strings.ToLower(name)]; ok {
		return c.maskFn(name, value)
	}
	return value
}

func (c *CurlRT) buildCurl(req *http.Request, body []byte) string {
	var b bytes.Buffer
	b.WriteString("curl -X ")
	b.WriteString(sanitizeLogValue(req.Method))
	b.WriteString(" '")
	b.WriteString(sanitizeLogValue(req.URL.String()))
	b.WriteString("'")
	for k, vs := range req.Header {
		for _, v := range vs {
			v = c.mask(k, v)
			b.WriteString(" -H '")
			b.WriteString(sanitizeLogValue(k))
			b.WriteString(": ")
			b.WriteString(sanitizeLogValue(v))
			b.WriteString("'")
		}
	}
	if len(body) > 0 {
		b.WriteString(" --data '")
		b.WriteString(sanitizeLogValue(string(body)))
		b.WriteString("'")
	}
	return b.String()
}

func (c *CurlRT) formatHeaders(h http.Header) string {
	var b bytes.Buffer
	for k, vs := range h {
		for _, v := range vs {
			v = c.mask(k, v)
			b.WriteString(sanitizeLogValue(k))
			b.WriteString(": ")
			b.WriteString(sanitizeLogValue(v))
			b.WriteString("\n")
		}
	}
	return b.String()
}

type loggingReadCloser struct {
	rc      io.ReadCloser
	ctx     context.Context
	l       Logger
	cl      CtxLogger
	cap     int
	total   int
	summary *bytes.Buffer
}

func newLoggingReadCloser(rc io.ReadCloser, ctx context.Context, c *CurlRT) io.ReadCloser {
	var buf *bytes.Buffer
	if c.ctxLogger == nil {
		buf = &bytes.Buffer{}
	}
	ca := c.maxStreamLogBytes
	if ca <= 0 {
		ca = 8192
	}
	return &loggingReadCloser{rc: rc, ctx: ctx, l: c.logger, cl: c.ctxLogger, cap: ca, summary: buf}
}

func (lrc *loggingReadCloser) Read(p []byte) (int, error) {
	n, err := lrc.rc.Read(p)
	if n > 0 {
		chunk := p[:n]
		lines := bytes.Split(chunk, []byte("\n"))
		for i, line := range lines {
			if i < len(lines)-1 || len(line) > 0 {
				if lrc.cl != nil {
					lrc.cl.Printf(lrc.ctx, "[curl stream chunk] %s", string(line))
				} else {
					remaining := lrc.cap - lrc.total
					if remaining > 0 {
						toWrite := line
						if len(toWrite) > remaining {
							toWrite = toWrite[:remaining]
						}
						lrc.summary.Write(toWrite)
						lrc.summary.WriteByte('\n')
						lrc.total += len(toWrite)
					}
				}
			}
		}
	}
	return n, err
}

func (lrc *loggingReadCloser) Close() error {
	if lrc.summary != nil && lrc.summary.Len() > 0 {
		lrc.l.Printf("[curl stream summary]\n%s", lrc.summary.String())
	}
	return lrc.rc.Close()
}

type IDCtxLogger struct{ L Logger }

func (i IDCtxLogger) Printf(ctx context.Context, format string, args ...any) {
	v := ctx.Value("log_id")
	if s, ok := v.(string); ok && s != "" {
		i.L.Printf("[req_id=%s] "+format, append([]any{s}, args...)...)
		return
	}
	i.L.Printf(format, args...)
}
