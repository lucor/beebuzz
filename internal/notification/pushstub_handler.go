package notification

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"lucor.dev/beebuzz/internal/core"
)

const pushStubLongPollTimeout = 30 * time.Second

// NewPushStubHandler returns a handler that long-polls the push stub broker for
// captured push events. It is intended for local development and test flows only
// and must never be exposed in production.
//
// The handler rejects non-loopback clients as defense-in-depth.
func NewPushStubHandler(broker *PushStubBroker, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r) {
			log.Warn("push stub request from non-loopback address rejected", "remote_addr", r.RemoteAddr)
			core.WriteForbidden(w, "access_denied", "push stub is only available from loopback")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), pushStubLongPollTimeout)
		defer cancel()

		ev, err := broker.Next(ctx)
		if err != nil {
			// Next only returns ctx.Err(), so a non-nil error here is always a
			// timeout or client cancellation. Long-poll convention: 204 means
			// "no event yet, retry".
			w.WriteHeader(http.StatusNoContent)
			return
		}

		core.WriteJSON(w, http.StatusOK, ev)
	}
}

// isLoopback reports whether the request originates from a loopback address.
func isLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr may not contain a port in some test setups.
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
