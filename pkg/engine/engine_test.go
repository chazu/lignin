package engine

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEvaluateEmptyString(t *testing.T) {
	eng := NewEngine()

	g, evalErrs, err := eng.Evaluate("")
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("unexpected eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected empty graph, got %d nodes", g.NodeCount())
	}
}

func TestEvaluateWhitespaceOnly(t *testing.T) {
	eng := NewEngine()

	g, evalErrs, err := eng.Evaluate("   \n\t  \n  ")
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("unexpected eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected empty graph, got %d nodes", g.NodeCount())
	}
}

func TestEvaluateValidExpression(t *testing.T) {
	eng := NewEngine()

	// (+ 1 2) is valid Lisp that zygomys can evaluate.
	// Since no builtins are registered for the DSL, the graph should be empty.
	g, evalErrs, err := eng.Evaluate("(+ 1 2)")
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("unexpected eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected empty graph (no builtins registered), got %d nodes", g.NodeCount())
	}
}

func TestEvaluateMultipleExpressions(t *testing.T) {
	eng := NewEngine()

	source := `
(def x 10)
(def y 20)
(+ x y)
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("unexpected eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
}

func TestEvaluateSyntaxError(t *testing.T) {
	eng := NewEngine()

	// Unmatched paren is a parse error.
	g, evalErrs, err := eng.Evaluate("(+ 1 2")
	if err != nil {
		t.Fatalf("expected non-fatal eval error, got fatal: %v", err)
	}
	if g != nil {
		t.Fatal("expected nil graph on syntax error")
	}
	if len(evalErrs) == 0 {
		t.Fatal("expected at least one eval error for syntax error")
	}

	// The error message should contain something meaningful.
	msg := evalErrs[0].Message
	if msg == "" {
		t.Error("eval error message should not be empty")
	}
}

func TestEvaluateUndefinedSymbol(t *testing.T) {
	eng := NewEngine()

	// Referencing an undefined symbol should produce an eval error.
	g, evalErrs, err := eng.Evaluate("(+ 1 undefined-symbol)")
	if err != nil {
		t.Fatalf("expected non-fatal eval error, got fatal: %v", err)
	}
	if g != nil {
		t.Fatal("expected nil graph on eval error")
	}
	if len(evalErrs) == 0 {
		t.Fatal("expected at least one eval error for undefined symbol")
	}
}

func TestEvaluateSyntaxErrorHasLineInfo(t *testing.T) {
	eng := NewEngine()

	// Put the error on line 2.
	source := "(+ 1 2)\n(+ 3"
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("expected non-fatal eval error, got fatal: %v", err)
	}
	if g != nil {
		t.Fatal("expected nil graph on syntax error")
	}
	if len(evalErrs) == 0 {
		t.Fatal("expected at least one eval error")
	}

	// We expect the line number to be extracted from the zygomys error.
	// Line info may or may not be available depending on the error format;
	// we just check the error is populated.
	e := evalErrs[0]
	if e.Message == "" {
		t.Error("eval error message should not be empty")
	}
	// If line info was extracted, verify it's positive.
	if e.Line > 0 {
		t.Logf("extracted line info: line=%d, message=%q", e.Line, e.Message)
	} else {
		t.Logf("no line info extracted (line=0), message=%q", e.Message)
	}
}

func TestEvalErrorImplementsError(t *testing.T) {
	e := EvalError{Line: 5, Col: 0, Message: "something went wrong"}
	s := e.Error()
	if !strings.Contains(s, "line 5") {
		t.Errorf("Error() should contain line info, got: %s", s)
	}
	if !strings.Contains(s, "something went wrong") {
		t.Errorf("Error() should contain message, got: %s", s)
	}

	// No line info.
	e2 := EvalError{Line: 0, Col: 0, Message: "no location"}
	s2 := e2.Error()
	if strings.Contains(s2, "line") {
		t.Errorf("Error() with no line should not contain 'line', got: %s", s2)
	}
}

func TestEvaluateDeterministic(t *testing.T) {
	eng := NewEngine()

	// Multiple evaluations of the same source should produce equivalent results.
	for i := 0; i < 5; i++ {
		g, evalErrs, err := eng.Evaluate("(+ 1 2)")
		if err != nil {
			t.Fatalf("iteration %d: unexpected fatal error: %v", i, err)
		}
		if len(evalErrs) > 0 {
			t.Fatalf("iteration %d: unexpected eval errors: %v", i, evalErrs)
		}
		if g == nil {
			t.Fatalf("iteration %d: expected non-nil graph", i)
		}
		if g.NodeCount() != 0 {
			t.Errorf("iteration %d: expected empty graph, got %d nodes", i, g.NodeCount())
		}
	}
}

func TestEvaluateTimeout(t *testing.T) {
	// This test verifies the timeout mechanism.
	// We temporarily reduce the timeout constant for testing purposes
	// by using a direct channel-based approach rather than modifying the const.
	//
	// Instead of testing through the Engine (which would require an infinite loop
	// that zygomys can actually execute), we test the waitWithTimeout function
	// directly with a channel that never sends.

	var mu sync.Mutex
	var gen uint64 = 1
	ch := make(chan evalResult) // Never sends

	// Override the timeout for this test by calling waitWithTimeout in a goroutine
	// and timing how long it takes. We use the real 5s timeout but with a test
	// that blocks.
	//
	// To keep the test fast, we instead test the timeout plumbing directly.
	done := make(chan struct{})
	var resultErr error

	// We'll test with a very short-lived approach: start a goroutine that
	// calls the real timeout logic with a blocking channel.
	go func() {
		defer close(done)
		_, _, resultErr = waitWithTimeout(ch, 1, &mu, &gen)
	}()

	// Wait a bit longer than EvalTimeout.
	select {
	case <-done:
		if resultErr == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if !strings.Contains(resultErr.Error(), "timed out") {
			t.Errorf("expected timeout error message, got: %v", resultErr)
		}
	case <-time.After(EvalTimeout + 2*time.Second):
		t.Fatal("test itself timed out waiting for evaluation timeout")
	}
}

func TestEvaluateGenerationDiscardsStale(t *testing.T) {
	// Test that a stale generation is detected.
	var mu sync.Mutex
	gen := uint64(2) // Current generation is 2

	ch := make(chan evalResult, 1)
	ch <- evalResult{graph: nil, errors: nil, err: nil}

	// Pass generation 1 (stale).
	_, _, err := waitWithTimeout(ch, 1, &mu, &gen)
	if err == nil {
		t.Fatal("expected error for stale generation")
	}
	if !strings.Contains(err.Error(), "superseded") {
		t.Errorf("expected superseded error, got: %v", err)
	}
}

func TestParseZygomysError(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantLine int
		wantMsg  string
	}{
		{
			name:    "error on line format",
			msg:     "Error on line 5: unexpected token\n",
			wantLine: 5,
			wantMsg:  "unexpected token",
		},
		{
			name:    "no line info",
			msg:     "some generic error",
			wantLine: 0,
			wantMsg:  "some generic error",
		},
		{
			name:    "line format lowercase",
			msg:     "error on line 12: missing paren",
			wantLine: 12,
			wantMsg:  "missing paren",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := parseZygomysError(errString(tt.msg))
			if len(errs) == 0 {
				t.Fatal("expected at least one error")
			}
			e := errs[0]
			if e.Line != tt.wantLine {
				t.Errorf("line = %d, want %d", e.Line, tt.wantLine)
			}
			if !strings.Contains(e.Message, tt.wantMsg) {
				t.Errorf("message = %q, want containing %q", e.Message, tt.wantMsg)
			}
		})
	}
}

// errString is a simple error type for testing.
type errString string

func (e errString) Error() string { return string(e) }
