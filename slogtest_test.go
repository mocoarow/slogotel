package slogotel_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/require"

	"github.com/mocoarow/slogotel"
)

func Test_Handler_slogtest(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	h := slogotel.New(slog.NewJSONHandler(&buf, nil))

	results := func() []map[string]any {
		var ms []map[string]any
		for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal(line, &m))
			ms = append(ms, m)
		}
		return ms
	}

	require.NoError(t, slogtest.TestHandler(h, results))
}
