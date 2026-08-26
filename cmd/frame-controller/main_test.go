package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestRunWritesSuccessJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"capabilities"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true || len(result["capabilities"].([]any)) < 8 || stderr.Len() != 0 {
		t.Fatalf("stdout=%s stderr=%s", stdout.Bytes(), stderr.Bytes())
	}
}

func TestRunWritesPublicErrorJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit=%d", code)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != false || result["error"] == "" || stderr.Len() != 0 {
		t.Fatalf("stdout=%s stderr=%s", stdout.Bytes(), stderr.Bytes())
	}
}

func TestRunReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"capabilities"}, failingWriter{}, &stderr); code != 1 {
		t.Fatalf("exit=%d", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("missing encoder error")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
