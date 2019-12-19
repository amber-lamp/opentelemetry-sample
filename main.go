package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
	"math/rand"

	"go.opentelemetry.io/otel/api/distributedcontext"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"

	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() func() {
	// Create Jaeger Exporter
	exporter, err := jaeger.NewExporter(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "opentelemetry-sample",
			Tags: []core.KeyValue{
				key.String("exporter", "jaeger"),
				key.Float64("float", 312.23),
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
	return func() {
		exporter.Flush()
	}
}

func main() {
	fn := initTracer()
	defer fn()

	tr := global.TraceProvider().Tracer("component-main")
	fortuneHandler := func(w http.ResponseWriter, req *http.Request) {
		attrs, entries, spanCtx := httptrace.Extract(req.Context(), req)

		req = req.WithContext(distributedcontext.WithMap(req.Context(), distributedcontext.NewMap(distributedcontext.MapUpdate{
			MultiKV: entries,
		})))

		ctx, span := tr.Start(
			req.Context(),
			"fortune",
			trace.WithAttributes(attrs...),
			trace.ChildOf(spanCtx),
		)
		defer span.End()

		span.AddEvent(ctx, "handling this...")
		omikuji := omikuji(ctx)

		_, _ = io.WriteString(w, "運勢は" + omikuji + "です")
	}

	http.HandleFunc("/fortune", fortuneHandler)
	err := http.ListenAndServe(":7777", nil)
	if err != nil {
		panic(err)
	}
}
func omikuji(ctx context.Context) string {
	tr := global.TraceProvider().Tracer("component-omikuji")
	_, span := tr.Start(ctx, "omikuji")
	defer span.End()
	t := time.Now()
	var omikuji string
	var msg string
	if (t.Month() == 1 && t.Day() >= 1 && t.Day() <= 3){
		omikuji = "大吉"
		msg = "お正月は大吉"
	} else {
		t := t.UnixNano()
		rand.Seed(t)
		s := rand.Intn(6)
		switch s {
		case 0:
			omikuji = "凶"
			msg = "残念でした"
			time.Sleep(time.Second)
		case 1, 2:
			omikuji = "吉"
			msg = "そこそこでした"
		case 3, 4:
			omikuji = "中吉"
			msg = "まあまあでした"
		case 5:
			omikuji = "大吉"
			msg = "いいですね"
		}
	}
	span.AddEvent(ctx,msg)
	return omikuji
}