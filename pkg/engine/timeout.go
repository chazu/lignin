package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/chazu/lignin/pkg/graph"
)

// EvalTimeout is the hard limit for a single evaluation.
const EvalTimeout = 5 * time.Second

// result is the internal type used to pass evaluation results through channels.
type evalResult struct {
	graph  *graph.DesignGraph
	errors []EvalError
	err    error
}

// waitWithTimeout waits for a result from ch, but returns a timeout error
// if the evaluation exceeds EvalTimeout. It uses a generation counter to
// discard stale results from previous evaluations.
//
// On timeout, the goroutine may still be running; the generation check
// ensures its result is discarded when it eventually completes.
func waitWithTimeout(
	ch <-chan evalResult,
	gen uint64,
	mu *sync.Mutex,
	currentGen *uint64,
) (*graph.DesignGraph, []EvalError, error) {
	timer := time.NewTimer(EvalTimeout)
	defer timer.Stop()

	select {
	case res := <-ch:
		// Check if this result is still relevant (not stale).
		mu.Lock()
		current := *currentGen
		mu.Unlock()

		if gen != current {
			// A newer evaluation was started; discard this result.
			return nil, nil, fmt.Errorf("evaluation superseded by newer request")
		}

		return res.graph, res.errors, res.err

	case <-timer.C:
		return nil, nil, fmt.Errorf("evaluation timed out after %s", EvalTimeout)
	}
}
