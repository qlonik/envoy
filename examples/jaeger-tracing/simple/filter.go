package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

var UpdateUpstreamBody = "upstream response body updated by the simple plugin"

// The callbacks in the filter, like `DecodeHeaders`, can be implemented on demand.
// Because api.PassThroughStreamFilter provides a default implementation.
type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	path      string
	config    *config
}

// based on https://github.com/openzipkin/zipkin-go/blob/e84b2cf6d2d915fe0ee57c2dc4d736ec13a2ef6a/propagation/b3/http.go#L53
func extractParentContextFromHeaders(h api.HeaderMap) (*model.SpanContext, error) {
	var (
		traceIDHeader, _      = h.Get(b3.TraceID)
		spanIDHeader, _       = h.Get(b3.SpanID)
		parentSpanIDHeader, _ = h.Get(b3.ParentSpanID)
		sampledHeader, _      = h.Get(b3.Sampled)
		flagsHeader, _        = h.Get(b3.Flags)
		singleHeader, _       = h.Get(b3.Context)
	)

	var (
		sc   *model.SpanContext
		sErr error
		mErr error
	)
	if singleHeader != "" {
		sc, sErr = b3.ParseSingleHeader(singleHeader)
		if sErr == nil {
			return sc, nil
		}
	}

	sc, mErr = b3.ParseHeaders(
		traceIDHeader, spanIDHeader, parentSpanIDHeader,
		sampledHeader, flagsHeader,
	)

	if mErr != nil && sErr != nil {
		return nil, sErr
	}

	return sc, mErr
}

// based on https://github.com/openzipkin/zipkin-go/blob/e84b2cf6d2d915fe0ee57c2dc4d736ec13a2ef6a/propagation/b3/http.go#L90
func injectParentcontextIntoRequestHeaders(h *api.RequestHeaderMap, sc model.SpanContext) error {
	if h == nil {
		return errors.New("missing target header map")
	}

	if (model.SpanContext{}) == sc {
		return b3.ErrEmptyContext
	}

	if sc.Debug {
		(*h).Set(b3.Flags, "1")
	} else if sc.Sampled != nil {
		// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
		// so don't also send "X-B3-Sampled: 1".
		if *sc.Sampled {
			(*h).Set(b3.Sampled, "1")
		} else {
			(*h).Set(b3.Sampled, "0")
		}
	}

	if !sc.TraceID.Empty() && sc.ID > 0 {
		(*h).Set(b3.TraceID, sc.TraceID.String())
		(*h).Set(b3.SpanID, sc.ID.String())
		if sc.ParentID != nil {
			(*h).Set(b3.ParentSpanID, sc.ParentID.String())
		}
	}

	return nil
}

func injectParentcontextIntoResponseHeaders(h *api.ResponseHeaderMap, sc model.SpanContext) error {
	if h == nil {
		return errors.New("missing target header map")
	}

	if (model.SpanContext{}) == sc {
		return b3.ErrEmptyContext
	}

	if sc.Debug {
		(*h).Set(b3.Flags, "1")
	} else if sc.Sampled != nil {
		// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
		// so don't also send "X-B3-Sampled: 1".
		if *sc.Sampled {
			(*h).Set(b3.Sampled, "1")
		} else {
			(*h).Set(b3.Sampled, "0")
		}
	}

	if !sc.TraceID.Empty() && sc.ID > 0 {
		(*h).Set(b3.TraceID, sc.TraceID.String())
		(*h).Set(b3.SpanID, sc.ID.String())
		if sc.ParentID != nil {
			(*h).Set(b3.ParentSpanID, sc.ParentID.String())
		}
	}

	return nil
}

func (f *filter) sendLocalReplyInternal() api.StatusType {
	body := fmt.Sprintf("%s, path: %s\r\n", f.config.echoBody, f.path)
	f.callbacks.SendLocalReply(200, body, nil, 0, "")
	// Remember to return LocalReply when the request is replied locally
	return api.LocalReply
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	// set up a span reporter
	reporter := httpreporter.NewReporter(
		"http://jaeger:9411/api/v2/spans",
		// forces sending spans right away
		httpreporter.BatchSize(0),
	)
	defer func() { _ = reporter.Close() }()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("golang-filter", "")
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	log.Println("+++")
	header.Range(func(key, value string) bool {
		if key == "b3" || strings.HasPrefix(key, "x-b3") {
			log.Printf("%s: \"%s\"", key, value)
		}
		return true
	})
	log.Println("---")

	parentContext := tracer.Extract(func() (*model.SpanContext, error) {
		return extractParentContextFromHeaders(header)
	})

	span := tracer.StartSpan("test span in decode headers", zipkin.Parent(parentContext))
	defer span.Finish()

	span.Tag("test-tag", "test-value")
	span.Tag("direction", f.config.direction)

	_ = injectParentcontextIntoRequestHeaders(&header, span.Context())

	f.path, _ = header.Get(":path")
	api.LogDebugf("get path %s", f.path)

	if f.path == "/localreply_by_config" {
		return f.sendLocalReplyInternal()
	}
	return api.Continue
	/*
		// If the code is time-consuming, to avoid blocking the Envoy,
		// we need to run the code in a background goroutine
		// and suspend & resume the filter
		go func() {
			defer f.callbacks.RecoverPanic()
			// do time-consuming jobs

			// resume the filter
			f.callbacks.Continue(status)
		}()

		// suspend the filter
		return api.Running
	*/
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

// Callbacks which are called in response path
// The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	// set up a span reporter
	reporter := httpreporter.NewReporter(
		"http://jaeger:9411/api/v2/spans",
		// forces sending spans right away
		httpreporter.BatchSize(0),
	)
	defer func() { _ = reporter.Close() }()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("golang-filter", "")
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	// TODO: this does not extract any context. it should be shared from `DecodeHeaders` step
	parentContext := tracer.Extract(func() (*model.SpanContext, error) {
		return extractParentContextFromHeaders(header)
	})

	span := tracer.StartSpan("test span in encode headers", zipkin.Parent(parentContext))
	defer span.Finish()

	span.Tag("test-tag2", "test-value2")
	span.Tag("direction", f.config.direction)

	_ = injectParentcontextIntoResponseHeaders(&header, span.Context())

	if f.path == "/update_upstream_response" {
		header.Set("Content-Length", strconv.Itoa(len(UpdateUpstreamBody)))
	}
	header.Set("Rsp-Header-From-Go", "bar-test")
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

// EncodeData might be called multiple times during handling the response body.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if f.path == "/update_upstream_response" {
		if endStream {
			buffer.SetString(UpdateUpstreamBody)
		} else {
			buffer.Reset()
		}
	}
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

// OnLog is called when the HTTP stream is ended on HTTP Connection Manager filter.
func (f *filter) OnLog() {
	code, _ := f.callbacks.StreamInfo().ResponseCode()
	respCode := strconv.Itoa(int(code))
	api.LogDebug(respCode)

	/*
		// It's possible to kick off a goroutine here.
		// But it's unsafe to access the f.callbacks because the FilterCallbackHandler
		// may be already released when the goroutine is scheduled.
		go func() {
			defer func() {
				if p := recover(); p != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					fmt.Printf("http: panic serving: %v\n%s", p, buf)
				}
			}()

			// do time-consuming jobs
		}()
	*/
}

// OnLogDownstreamStart is called when HTTP Connection Manager filter receives a new HTTP request
// (required the corresponding access log type is enabled)
func (f *filter) OnLogDownstreamStart() {
	// also support kicking off a goroutine here, like OnLog.
}

// OnLogDownstreamPeriodic is called on any HTTP Connection Manager periodic log record
// (required the corresponding access log type is enabled)
func (f *filter) OnLogDownstreamPeriodic() {
	// also support kicking off a goroutine here, like OnLog.
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
	// One should not access f.callbacks here because the FilterCallbackHandler
	// is released. But we can still access other Go fields in the filter f.

	// goroutine can be used everywhere.
}
