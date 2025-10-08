package tracing

import "context"

// Attribute represents a key/value pair attached to a span.
type Attribute struct {
	Key   string
	Value any
}

// Span represents an in-flight tracing span.
type Span interface {
	End(err error)
}

// Tracer starts spans for tracing operations.
type Tracer interface {
	Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span)
}

// NoopTracer discards all tracing events.
type NoopTracer struct{}

// Start implements Tracer.
func (NoopTracer) Start(ctx context.Context, _ string, _ ...Attribute) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

// End implements Span.
func (noopSpan) End(error) {}

// compositeTracer fans out spans to multiple tracers.
type compositeTracer struct {
	primary Tracer
	rest    []Tracer
}

func (c compositeTracer) Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span) {
	if c.primary == nil {
		for i, tracer := range c.rest {
			if tracer != nil {
				c.primary = tracer
				c.rest = append(append([]Tracer(nil), c.rest[:i]...), c.rest[i+1:]...)
				break
			}
		}
		if c.primary == nil {
			return ctx, noopSpan{}
		}
	}

	ctxPrimary, primarySpan := c.primary.Start(ctx, name, attrs...)
	if primarySpan == nil {
		primarySpan = noopSpan{}
	}
	spans := []Span{primarySpan}

	for _, tracer := range c.rest {
		if tracer == nil {
			continue
		}
		_, span := tracer.Start(ctx, name, attrs...)
		if span == nil {
			span = noopSpan{}
		}
		spans = append(spans, span)
	}

	return ctxPrimary, compositeSpan(spans)
}

type compositeSpan []Span

func (cs compositeSpan) End(err error) {
	for _, span := range cs {
		span.End(err)
	}
}

// WithTracer normalises the provided tracer set, returning a composite tracer.
func WithTracer(primary Tracer, others ...Tracer) Tracer {
	tracers := make([]Tracer, 0, 1+len(others))
	if primary != nil {
		tracers = append(tracers, primary)
	}
	for _, t := range others {
		if t != nil {
			tracers = append(tracers, t)
		}
	}
	if len(tracers) == 0 {
		return NoopTracer{}
	}
	if len(tracers) == 1 {
		return tracers[0]
	}
	return compositeTracer{primary: tracers[0], rest: tracers[1:]}
}

// String attribute helper.
func String(key, value string) Attribute { return Attribute{Key: key, Value: value} }

// Int attribute helper.
func Int(key string, value int) Attribute { return Attribute{Key: key, Value: value} }

// Bool attribute helper.
func Bool(key string, value bool) Attribute { return Attribute{Key: key, Value: value} }
