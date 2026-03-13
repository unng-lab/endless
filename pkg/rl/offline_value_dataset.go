package rl

import "fmt"

// DiscountedReturnExample keeps one already-vectorized transition together with its discounted
// return-to-go target. External trainers can reuse this instead of rebuilding episode continuity
// logic around the stable transition tensor contract.
type DiscountedReturnExample struct {
	Transition       VectorizedTransition
	DiscountedReturn float32
}

// DiscountedReturnSummary reports the basic continuity and target-range properties of the
// discounted-return dataset derived from one transition slice.
type DiscountedReturnSummary struct {
	Samples              int
	ContinuedTransitions int
	TerminalTransitions  int
	UnlinkedTransitions  int
	DiscountedReturnMin  float32
	DiscountedReturnMax  float32
}

// BuildDiscountedReturnExamples reconstructs per-episode return-to-go targets from already
// vectorized transition rows. The function assumes rows stay ordered by episode_id/tick, exactly
// like the canonical transitions view and JSONL export currently emit them.
func BuildDiscountedReturnExamples(transitions []VectorizedTransition, discount float32) ([]DiscountedReturnExample, DiscountedReturnSummary, error) {
	if len(transitions) == 0 {
		return nil, DiscountedReturnSummary{}, fmt.Errorf("transition dataset is empty")
	}

	discount = clampFloat32(discount, 0, 1)
	examples := make([]DiscountedReturnExample, len(transitions))
	summary := DiscountedReturnSummary{}

	for index, transition := range transitions {
		examples[index] = DiscountedReturnExample{
			Transition:       transition,
			DiscountedReturn: transition.Reward,
		}
		summary.Samples++
		if transition.Done >= 0.5 {
			summary.TerminalTransitions++
			continue
		}
		if index+1 < len(transitions) &&
			transitions[index+1].EpisodeID == transition.EpisodeID &&
			transitions[index+1].Tick == transition.Tick+1 {
			summary.ContinuedTransitions++
			continue
		}
		summary.UnlinkedTransitions++
	}

	for index := len(examples) - 1; index >= 0; index-- {
		if examples[index].Transition.Done >= 0.5 {
			continue
		}
		if index+1 >= len(examples) {
			continue
		}

		nextExample := examples[index+1]
		if nextExample.Transition.EpisodeID != examples[index].Transition.EpisodeID {
			continue
		}
		if nextExample.Transition.Tick != examples[index].Transition.Tick+1 {
			continue
		}
		examples[index].DiscountedReturn += discount * nextExample.DiscountedReturn
	}

	for index, example := range examples {
		if index == 0 {
			summary.DiscountedReturnMin = example.DiscountedReturn
			summary.DiscountedReturnMax = example.DiscountedReturn
			continue
		}
		summary.DiscountedReturnMin = minFloat32(summary.DiscountedReturnMin, example.DiscountedReturn)
		summary.DiscountedReturnMax = maxFloat32(summary.DiscountedReturnMax, example.DiscountedReturn)
	}
	return examples, summary, nil
}
