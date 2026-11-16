package verdict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountsTotal(t *testing.T) {
	tests := []struct {
		name     string
		counts   Counts
		expected int
	}{
		{
			name: "only completed",
			counts: Counts{
				Completed: 1,
			},
			expected: 1,
		},
		{
			name: "only target",
			counts: Counts{
				Target: 2,
			},
			expected: 2,
		},
		{
			name: "only other",
			counts: Counts{
				Other: 3,
			},
			expected: 3,
		},
		{
			name: "only boot failed",
			counts: Counts{
				BootFailed: 4,
			},
			expected: 4,
		},
		{
			name: "only trash",
			counts: Counts{
				Trash: 5,
			},
			expected: 5,
		},
		{
			name: "all fields",
			counts: Counts{
				Completed:  1,
				Target:     2,
				Other:      4,
				BootFailed: 8,
				Trash:      16,
			},
			expected: 31,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			got := test.counts.Total()
			assert.Equal(tt, test.expected, got)
		})
	}
}

func TestCountsAdd(t *testing.T) {
	tests := []struct {
		name     string
		input    Outcome
		expected Counts
	}{
		{
			name:  "add completed",
			input: OutcomeCompleted,
			expected: Counts{
				Completed: 1,
			},
		},
		{
			name:  "add target crash",
			input: OutcomeTargetCrash,
			expected: Counts{
				Target: 1,
			},
		},
		{
			name:  "add other crash",
			input: OutcomeOtherCrash,
			expected: Counts{
				Other: 1,
			},
		},
		{
			name:  "add boot failed",
			input: OutcomeBootFailed,
			expected: Counts{
				BootFailed: 1,
			},
		},
		{
			name:  "add trash",
			input: OutcomeTrash,
			expected: Counts{
				Trash: 1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			got := Counts{}
			got.Add(test.input)
			assert.Equal(tt, test.expected, got)
		})
	}
}

func TestCountsAddConcurrent(t *testing.T) {
	outcomes := []Outcome{
		OutcomeCompleted,
		OutcomeCompleted,
		OutcomeTargetCrash,
		OutcomeOtherCrash,
		OutcomeBootFailed,
		OutcomeTrash,
		OutcomeTrash,
	}
	var c Counts
	for _, outcome := range outcomes {
		c.Add(outcome)
	}

	assert.Equal(t, Counts{
		Completed:  2,
		Target:     1,
		Other:      1,
		BootFailed: 1,
		Trash:      2,
	}, c)
	assert.Equal(t, 7, c.Total())
}

func TestOutcomeString(t *testing.T) {
	tests := []struct {
		name     string
		input    Outcome
		expected string
	}{
		{"completed", OutcomeCompleted, "completed"},
		{"target crash", OutcomeTargetCrash, "target-crash"},
		{"other crash", OutcomeOtherCrash, "other-crash"},
		{"boot failed", OutcomeBootFailed, "boot-failed"},
		{"trash", OutcomeTrash, "trash"},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			assert.Equal(tt, test.expected, test.input.String())
		})
	}
}

func TestOutcomeIsCrash(t *testing.T) {
	tests := []struct {
		name     string
		input    Outcome
		expected bool
	}{
		{"completed", OutcomeCompleted, false},
		{"target crash", OutcomeTargetCrash, true},
		{"other crash", OutcomeOtherCrash, true},
		{"boot failed", OutcomeBootFailed, false},
		{"trash", OutcomeTrash, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			assert.Equal(tt, test.expected, test.input.IsCrash())
		})
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		counts   Counts
		required int
		expected Result
	}{
		{
			name:     "completed",
			counts:   Counts{Completed: 1},
			required: 1,
			expected: Result{
				Kind:     NotReproduced,
				Reason:   "1/1 runs completed with no target crash",
				Required: 1,
			},
		},
		{
			name:     "more completed than required",
			counts:   Counts{Completed: 5},
			required: 3,
			expected: Result{
				Kind:     NotReproduced,
				Reason:   "5/3 runs completed with no target crash",
				Required: 3,
			},
		},
		{
			name:     "target crash",
			counts:   Counts{Target: 1},
			required: 1,
			expected: Result{
				Kind:     Reproduced,
				Reason:   "target crash observed on run 1 of 1 (completed: 0, other crashes: 0, trash: 0)",
				Required: 1,
			},
		},
		{
			name:     "target crash wins over completed",
			counts:   Counts{Completed: 5, Target: 1},
			required: 5,
			expected: Result{
				Kind:     Reproduced,
				Reason:   "target crash observed on run 6 of 6 (completed: 5, other crashes: 0, trash: 0)",
				Required: 5,
			},
		},
		{
			name:     "other crash with insufficient completed",
			counts:   Counts{Other: 1},
			required: 1,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "only 0/1 clean runs completed, check was interrupted",
				Required: 1,
			},
		},
		{
			name:     "other crashes but enough completed",
			counts:   Counts{Completed: 2, Other: 3},
			required: 2,
			expected: Result{
				Kind:     NotReproduced,
				Reason:   "2/2 runs completed with no target crash",
				Required: 2,
			},
		},
		{
			name:     "boot failed",
			counts:   Counts{BootFailed: 2},
			required: 3,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "kernel does not boot (2/2 attempts); fix the kernel first",
				Required: 3,
			},
		},
		{
			name:     "boot failed takes precedence",
			counts:   Counts{BootFailed: 1, Trash: 1, Other: 1},
			required: 5,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "kernel does not boot (1/3 attempts); fix the kernel first",
				Required: 5,
			},
		},
		{
			name:     "trash only",
			counts:   Counts{Trash: 4},
			required: 3,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "no usable runs: 4/4 attempts failed on infrastructure",
				Required: 3,
			},
		},
		{
			name:     "some completed plus trash",
			counts:   Counts{Completed: 1, Trash: 2},
			required: 3,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "only 1/3 clean runs completed, 2 attempts were unusable",
				Required: 3,
			},
		},
		{
			name:     "zero required",
			counts:   Counts{},
			required: 0,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "required runs must be positive",
				Required: 0,
			},
		},
		{
			name:     "negative required",
			counts:   Counts{},
			required: -1,
			expected: Result{
				Kind:     Inconclusive,
				Reason:   "required runs must be positive",
				Required: -1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			got := test.counts.Decide(test.required)
			assert.Equal(tt, test.expected, got)
		})
	}
}
