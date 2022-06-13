package provider

import (
	"bytes"
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type LogWriter struct {
	Context context.Context
}

func (w *LogWriter) Write(buf []byte) (int, error) {
	lines := bytes.Split(buf, []byte{'\n'})
	for _, line := range lines {
		tflog.Info(w.Context, string(line))
	}

	return len(buf), nil
}

func NewLogWriter(ctx context.Context) *LogWriter {
	return &LogWriter{
		Context: ctx,
	}
}
