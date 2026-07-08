package ai

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestEmitNoticeNotifier(t *testing.T) {
	t.Parallel()

	var got []string
	ctx := WithNotifier(context.Background(), func(msg string) {
		got = append(got, msg)
	})
	emitNotice(ctx, "Modelo sobrecarregado — tentando novamente em 3s (1/3)...")

	if len(got) != 1 || got[0] != "Modelo sobrecarregado — tentando novamente em 3s (1/3)..." {
		t.Fatalf("notifier got %v", got)
	}
}

func TestEmitNoticeStderrFallback(t *testing.T) {
	t.Parallel()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- strings.TrimSpace(buf.String())
	}()

	emitNotice(context.Background(), "aviso de teste")
	_ = w.Close()
	os.Stderr = old

	out := <-done
	if out != "aviso de teste" {
		t.Fatalf("stderr = %q", out)
	}
}