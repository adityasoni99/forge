package blueprint

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// testRecordParallelApply, when non-nil, is invoked with each node ID as RunState.NodeResults
// is updated in declaration order after a parallel fan-out (tests only).
var testRecordParallelApply func(nodeID string)

type parallelOutcome struct {
	id     string
	node   Node
	result NodeResult
}

func (e *Engine) mergeParallelNext(outcomes []parallelOutcome) (string, error) {
	if len(outcomes) == 0 {
		return "", nil
	}
	var target string
	defined := false
	for _, o := range outcomes {
		nexts := e.resolveNextNodes(o.id, o.node, o.result)
		if len(nexts) > 1 {
			return "", fmt.Errorf("node %q: nested parallel fan-out is not supported", o.id)
		}
		if len(nexts) == 0 {
			if defined && target != "" {
				return "", fmt.Errorf("parallel branches disagree: mixed termination and merge targets")
			}
			target = ""
			defined = true
			continue
		}
		n := nexts[0]
		if !defined {
			target = n
			defined = true
			continue
		}
		if target == "" {
			return "", fmt.Errorf("parallel branches disagree: mixed termination and merge targets")
		}
		if target != n {
			return "", fmt.Errorf("parallel branches diverge: %q vs %q", target, n)
		}
	}
	return target, nil
}

func (e *Engine) runParallelFanOut(ctx context.Context, state *RunState, nextIDs []string) (string, error) {
	outcomes := make([]parallelOutcome, len(nextIDs))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(e.maxConcurrency)
	for i, nid := range nextIDs {
		i, nid := i, nid
		g.Go(func() error {
			node, ok := e.graph.GetNode(nid)
			if !ok {
				return fmt.Errorf("node %q not found", nid)
			}
			res, err := e.runNodeWithHooks(gctx, state, nid, node)
			if err != nil {
				return err
			}
			outcomes[i] = parallelOutcome{id: nid, node: node, result: res}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}
	for _, o := range outcomes {
		state.NodeResults[o.id] = o.result
		if testRecordParallelApply != nil {
			testRecordParallelApply(o.id)
		}
	}
	return e.mergeParallelNext(outcomes)
}
