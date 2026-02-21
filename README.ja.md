# slogotel

Go の `log/slog` と OpenTelemetry を連携する `slog.Handler` ラッパーライブラリ。

## 特徴

- **trace_id / span_id を常に付与** — Span の Recording 状態に依存せず、Valid な SpanContext があれば常にログ属性に追加
- **Span イベント記録** — Recording Span がある場合、ログメッセージを Span イベントとして記録
- **Baggage 付与** — Recording Span がある場合、Baggage メンバーをログ属性に追加
- **エラー時 Span Status 設定** — `slog.LevelError` 以上かつ Recording の場合、Span Status を Error に設定
- **Functional Options** — 各機能の有効/無効やキー名のカスタマイズが可能
- **slogtest.TestHandler 適合** — 標準の適合性テストに合格

## 背景

| ライブラリ | 課題 |
|-----------|------|
| `remychantenay/slog-otel` | `!span.IsRecording()` で早期リターンするため、NonRecording Span では trace_id が出力されない |
| `go-slog/otelslog` | trace_id/span_id 付与のみで Span イベント記録や Baggage 等の機能がない |

slogotel は両方のいいとこ取りをしたハンドラです。

## インストール

```sh
go get github.com/mocoarow/slogotel
```

## 使い方

### 基本

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mocoarow/slogotel"
)

func main() {
	ctx := context.Background()

	h := slogotel.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := slog.New(h)

	// context に Span があれば trace_id, span_id が自動付与される
	logger.InfoContext(ctx, "request received", "path", "/api/users")
}
```

出力例:

```json
{
  "time": "2024-01-01T00:00:00Z",
  "level": "INFO",
  "msg": "request received",
  "trace_id": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
  "span_id": "a1b2c3d4e5f6a1b2",
  "path": "/api/users"
}
```

### オプション付き

```go
h := slogotel.New(
	slog.NewJSONHandler(os.Stdout, nil),
	slogotel.WithSpanEvent(false),       // Span イベント記録を無効化
	slogotel.WithBaggage(false),         // Baggage 付与を無効化
	slogotel.WithTraceIDKey("traceId"),  // キー名カスタマイズ
	slogotel.WithSpanIDKey("spanId"),
	slogotel.WithTraceFlagsKey("traceFlags"),
)
```

## Options

| Option | デフォルト | 説明 |
|--------|-----------|------|
| `WithSpanEvent(bool)` | `true` | Recording Span にログメッセージを Span イベントとして記録 |
| `WithBaggage(bool)` | `true` | Recording Span で Baggage メンバーをログ属性に追加 |
| `WithTraceIDKey(string)` | `"trace_id"` | trace_id 属性のキー名 |
| `WithSpanIDKey(string)` | `"span_id"` | span_id 属性のキー名 |
| `WithTraceFlagsKey(string)` | `""` (無効) | trace_flags 属性のキー名（空文字で付与しない） |

## 処理フロー

```
Handle(ctx, record)
  │
  ├── 1. SpanContext 取得 ← Recording に依存しない
  │     ├── trace_id が Valid → 属性追加
  │     └── span_id が Valid → 属性追加
  │
  ├── 2. Span が Recording のときだけ:
  │     ├── Baggage メンバーをログ属性に追加
  │     ├── ログを Span イベントとして記録
  │     └── LevelError 以上で Span Status を Error に設定
  │
  └── 3. next.Handle(ctx, record)
```

## 注意事項

- Baggage のキーが既存のログ属性キーと衝突する場合、両方の値がレコードに追加されます（`slog.JSONHandler` 等のダウンストリームハンドラは通常、最後の値を使用します）
- Span イベントに記録される属性には `log.level` が自動付与されます
- Group 属性はドット区切りのフラットキーに展開されます（例: `req.method`）

## ライセンス

MIT
