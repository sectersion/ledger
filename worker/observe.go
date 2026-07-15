package worker

import (
	"context"
	"encoding/json"
	"strings"
)

// Sink receives every stream-json event a worker emits, keyed by agent ID
// — the M9 TUI's live journal-stream feed.
type Sink func(agentID string, e Event)

// Registrar is handed a worker's own cancel func the moment it starts, so
// something outside worker (the M9 TUI) can kill that specific worker
// later without touching any other running worker.
type Registrar func(agentID string, cancel context.CancelFunc)

// Relayer is consulted after a worker's run fails; if it returns a pending
// message for that agent ID (ok == true, consuming it), Run respawns the
// worker with the message prepended to its prompt instead of returning the
// error — the `/btw` relay's actual respawn.
type Relayer func(agentID string) (message string, ok bool)

type ctxKey int

const (
	agentIDKey ctxKey = iota
	sinkKey
	registrarKey
	relayerKey
)

// WithAgentID labels the next Run call with an explicit agent ID, instead
// of Run's default of deriving one from filepath.Base(cwd).
func WithAgentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, agentIDKey, id)
}

// WithSink attaches a Sink that every worker.Run call reached through ctx
// reports its events to.
func WithSink(ctx context.Context, sink Sink) context.Context {
	return context.WithValue(ctx, sinkKey, sink)
}

// WithRegistrar attaches a Registrar that every worker.Run call reached
// through ctx registers its cancel func with.
func WithRegistrar(ctx context.Context, r Registrar) context.Context {
	return context.WithValue(ctx, registrarKey, r)
}

// WithRelayer attaches a Relayer that every worker.Run call reached through
// ctx consults after a failed run, to support `/btw`'s respawn.
func WithRelayer(ctx context.Context, r Relayer) context.Context {
	return context.WithValue(ctx, relayerKey, r)
}

func idFromContext(ctx context.Context) string {
	id, _ := ctx.Value(agentIDKey).(string)
	return id
}

func sinkFromContext(ctx context.Context) Sink {
	s, _ := ctx.Value(sinkKey).(Sink)
	return s
}

func registrarFromContext(ctx context.Context) Registrar {
	r, _ := ctx.Value(registrarKey).(Registrar)
	return r
}

func relayerFromContext(ctx context.Context) Relayer {
	r, _ := ctx.Value(relayerKey).(Relayer)
	return r
}

// Summary renders a short human-readable line for an event, for a live
// journal-stream display. It returns "" for events not worth surfacing
// (system/init/hook noise).
func (e Event) Summary() string {
	switch e.Type {
	case "prompt":
		var p struct {
			Prompt string `json:"prompt"`
		}
		if err := json.Unmarshal(e.Raw, &p); err != nil {
			return ""
		}
		return "▶ " + p.Prompt
	case "assistant":
		var m struct {
			Message struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
					Name string `json:"name"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(e.Raw, &m); err != nil {
			return ""
		}
		var parts []string
		for _, c := range m.Message.Content {
			switch c.Type {
			case "text":
				parts = append(parts, c.Text)
			case "tool_use":
				parts = append(parts, "→ "+c.Name)
			}
		}
		return strings.Join(parts, " ")
	case "result":
		var r resultEvent
		if err := json.Unmarshal(e.Raw, &r); err != nil {
			return ""
		}
		if r.IsError {
			return "✗ " + r.Result
		}
		return "✓ " + r.Result
	default:
		return ""
	}
}
