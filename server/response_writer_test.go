// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nilHeaderResponseWriter behaves like the plugin SDK's writer after the
// server has canceled the request: Header returns nil and writes fail.
type nilHeaderResponseWriter struct{}

func (nilHeaderResponseWriter) Header() http.Header        { return nil }
func (nilHeaderResponseWriter) Write([]byte) (int, error)  { return 0, http.ErrHandlerTimeout }
func (nilHeaderResponseWriter) WriteHeader(statusCode int) {}

func TestHeaderSafeResponseWriter(t *testing.T) {
	t.Run("returns a writable header when the wrapped writer returns nil", func(t *testing.T) {
		w := newHeaderSafeResponseWriter(nilHeaderResponseWriter{})

		require.NotPanics(t, func() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		})
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("passes headers, status and body through to the wrapped writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		w := newHeaderSafeResponseWriter(rec)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("{}"))
		require.NoError(t, err)

		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "{}", rec.Body.String())
	})

	t.Run("passes Flush through to the wrapped writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		w := newHeaderSafeResponseWriter(rec)

		w.Flush()

		assert.True(t, rec.Flushed)
	})

	t.Run("does not panic on Flush when the wrapped writer can't flush", func(t *testing.T) {
		w := newHeaderSafeResponseWriter(nilHeaderResponseWriter{})

		assert.NotPanics(t, w.Flush)
	})

	t.Run("unwraps to the wrapped writer", func(t *testing.T) {
		rec := httptest.NewRecorder()

		assert.Same(t, rec, newHeaderSafeResponseWriter(rec).Unwrap())
	})
}
