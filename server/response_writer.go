// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
)

// headerSafeResponseWriter makes Header always return a writable map.
//
// The server cancels a plugin HTTP request when the client disconnects, and the
// plugin SDK's ResponseWriter.Header then returns nil. A handler that sets a
// header would panic and stop the plugin process. Core fixed this in
// mattermost/mattermost#40255; this wrapper can be removed once the plugin
// depends on a server/public release that includes it.
type headerSafeResponseWriter struct {
	http.ResponseWriter
	header http.Header
}

var _ http.Flusher = (*headerSafeResponseWriter)(nil)

func newHeaderSafeResponseWriter(w http.ResponseWriter) *headerSafeResponseWriter {
	return &headerSafeResponseWriter{ResponseWriter: w}
}

func (w *headerSafeResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = w.ResponseWriter.Header()
		if w.header == nil {
			w.header = make(http.Header)
		}
	}
	return w.header
}

// Flush passes through to the wrapped writer, which the plugin SDK's writer supports.
func (w *headerSafeResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the wrapped writer.
func (w *headerSafeResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
