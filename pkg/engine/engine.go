// Package engine provides the Lisp evaluation engine for Lignin.
// It wraps zygomys in a sandboxed environment and produces a DesignGraph
// from user source code.
package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/chazu/lignin/pkg/graph"
	zygo "github.com/glycerine/zygomys/zygo"
)

// EvalError represents a non-fatal error encountered during evaluation,
// such as a parse error or a runtime error in user code.
type EvalError struct {
	Line    int
	Col     int
	Message string
}

func (e EvalError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// EvalWarning represents a non-fatal warning produced during evaluation.
type EvalWarning struct {
	Line   int
	Col    int
	Message string
	NodeID graph.NodeID
}

// EvalResult bundles the full output of an evaluation for use by UI bindings.
type EvalResult struct {
	Graph    *graph.DesignGraph
	Errors   []EvalError
	Warnings []EvalWarning
}

// Engine wraps the zygomys interpreter for Lignin evaluation.
// It is safe for concurrent use; each call to Evaluate creates a fresh
// sandboxed environment for determinism.
type Engine struct {
	mu         sync.Mutex
	generation uint64
}

// NewEngine creates a new Engine instance.
func NewEngine() *Engine {
	return &Engine{}
}

// Evaluate takes Lisp source code and produces a new DesignGraph.
// Each call creates a fresh zygomys sandbox for deterministic evaluation.
//
// Return semantics:
//   - On success: returns graph + nil errors + nil error
//   - On parse/eval failure: returns nil graph + eval errors + nil error
//   - On fatal failure (timeout, panic): returns nil + nil + error
func (e *Engine) Evaluate(source string) (*graph.DesignGraph, []EvalError, error) {
	e.mu.Lock()
	e.generation++
	gen := e.generation
	e.mu.Unlock()

	ch := make(chan evalResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- evalResult{err: fmt.Errorf("panic during evaluation: %v", r)}
			}
		}()

		g, evalErrs, err := e.evaluate(source)
		ch <- evalResult{graph: g, errors: evalErrs, err: err}
	}()

	return waitWithTimeout(ch, gen, &e.mu, &e.generation)
}

// evaluate performs the actual zygomys evaluation in a fresh sandbox.
func (e *Engine) evaluate(source string) (*graph.DesignGraph, []EvalError, error) {
	// Empty source is a valid program that produces an empty graph.
	if strings.TrimSpace(source) == "" {
		return graph.New(), nil, nil
	}

	// Create a fresh sandboxed zygomys environment.
	// Sandbox mode prevents user code from accessing the filesystem or syscalls.
	env := zygo.NewZlispSandbox()
	defer env.Stop()

	// Load and compile the source string into bytecode.
	err := env.LoadString(source)
	if err != nil {
		evalErrs := parseZygomysError(err)
		return nil, evalErrs, nil
	}

	// Execute the compiled bytecode.
	_, err = env.Run()
	if err != nil {
		evalErrs := parseZygomysError(err)
		return nil, evalErrs, nil
	}

	// No builtins are registered yet, so the graph is always empty.
	// DSL builtins (board, joint, assembly, etc.) will populate the graph
	// in a subsequent task.
	return graph.New(), nil, nil
}

// linePattern matches zygomys error messages that include "Error on line N: ..."
var linePattern = regexp.MustCompile(`(?i)(?:error )?on line (\d+):\s*(.*)`)

// linePatternShort matches simpler "line N: ..." patterns.
var linePatternShort = regexp.MustCompile(`(?i)^line (\d+):\s*(.*)`)

// parseZygomysError converts a zygomys error into one or more EvalError values.
// It attempts to extract line number information from the error message.
func parseZygomysError(err error) []EvalError {
	msg := err.Error()

	// Try to extract line numbers from the error message.
	// zygomys formats parse errors as "Error on line N: <details>\n"
	if m := linePattern.FindStringSubmatch(msg); m != nil {
		line, _ := strconv.Atoi(m[1])
		detail := strings.TrimSpace(m[2])
		return []EvalError{{
			Line:    line,
			Col:     0,
			Message: detail,
		}}
	}

	if m := linePatternShort.FindStringSubmatch(msg); m != nil {
		line, _ := strconv.Atoi(m[1])
		detail := strings.TrimSpace(m[2])
		return []EvalError{{
			Line:    line,
			Col:     0,
			Message: detail,
		}}
	}

	// Fallback: no line info available.
	return []EvalError{{
		Line:    0,
		Col:     0,
		Message: strings.TrimSpace(msg),
	}}
}
