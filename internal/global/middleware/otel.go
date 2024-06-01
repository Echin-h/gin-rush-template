package middleware

import (
	"bytes"
	"gin-rush-template/internal/global/errs"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"io"
)

const (
	TracerKey = "otel-tracer"
)

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.GetTracerProvider().Tracer("gin-rush-template")
		spanName := c.Request.Method + " " + c.Request.URL.Path
		c.Set(TracerKey, tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()

		ctx := otel.GetTextMapPropagator().Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(
				attribute.KeyValue{Key: "http.method", Value: attribute.StringValue(c.Request.Method)},
				attribute.KeyValue{Key: "http.target", Value: attribute.StringValue(c.Request.URL.Path)},
				attribute.KeyValue{Key: "http.host", Value: attribute.StringValue(c.Request.Host)},
				attribute.KeyValue{Key: "http.flavor", Value: attribute.StringValue(c.Request.Proto)},
				attribute.KeyValue{Key: "net.peer.ip", Value: attribute.StringValue(c.ClientIP())},
				attribute.KeyValue{Key: "http.route", Value: attribute.StringValue(spanName)},
			),
		}

		ctx, span := tracer.Start(ctx, spanName, opts...)
		traceID := span.SpanContext().TraceID().String()
		c.Writer.Header().Set("X-Trace-ID", traceID)
		defer span.End()

		var body []byte
		if c.Request.Body != nil {
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(c.Request.Body)
			if err != nil {
				span.SetStatus(codes.Error, "Failed to read request body")
			}
			body = buf.Bytes()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}
		span.SetAttributes(attribute.String("http.request.body", string(body)))

		c.Request = c.Request.WithContext(ctx)
		c.Next()
		status := c.Writer.Status()
		span.SetAttributes(attribute.KeyValue{Key: "http.status_code", Value: attribute.Int64Value(int64(status))})
		if err, exists := c.Get(errs.ErrorContextKey); exists {
			e := err.(errs.Error)
			span.SetAttributes(
				attribute.KeyValue{Key: "error.code", Value: attribute.Int64Value(int64(e.Code))},
				attribute.KeyValue{Key: "error.message", Value: attribute.StringValue(e.Message)},
				attribute.KeyValue{Key: "error.origin", Value: attribute.StringValue(e.Origin)},
			)
			span.SetStatus(func() (code codes.Code) {
				if e.Code != 200 {
					return codes.Error
				}
				return codes.Ok
			}(), e.Message)
		} else {
			span.SetStatus(func() (code codes.Code) {
				if status != 200 {
					return codes.Error
				}
				return codes.Ok
			}(), "")
			if len(c.Errors) > 0 {
				span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
			}
		}
	}
}
