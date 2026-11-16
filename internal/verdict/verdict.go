package verdict

import (
	"fmt"
)

type Kind string

const (
	Reproduced    Kind = "reproduced"
	NotReproduced Kind = "not-reproduced"
	Inconclusive  Kind = "inconclusive"
)

type Result struct {
	Kind         Kind
	Reason       string
	Required     int
	OtherCrashes int
}

type Outcome int

const (
	// Program finished with no crashes
	OutcomeCompleted Outcome = iota
	OutcomeTargetCrash
	OutcomeOtherCrash
	OutcomeBootFailed
	OutcomeTrash
)

func (o Outcome) String() string {
	var result string
	switch o {
	case OutcomeCompleted:
		result = "completed"
	case OutcomeTargetCrash:
		result = "target-crash"
	case OutcomeOtherCrash:
		result = "other-crash"
	case OutcomeBootFailed:
		result = "boot-failed"
	case OutcomeTrash:
		result = "trash"
	}
	return result
}

func (o Outcome) IsCrash() bool {
	if o == OutcomeTargetCrash ||
		o == OutcomeOtherCrash {
		return true
	}
	return false
}

type Counts struct {
	Completed  int
	Target     int
	Other      int
	BootFailed int
	Trash      int
}

func (c Counts) Total() int {
	return c.Completed + c.Target + c.Other + c.BootFailed + c.Trash
}

func (c *Counts) Add(o Outcome) {
	switch o {
	case OutcomeCompleted:
		c.Completed++
	case OutcomeTargetCrash:
		c.Target++
	case OutcomeOtherCrash:
		c.Other++
	case OutcomeBootFailed:
		c.BootFailed++
	case OutcomeTrash:
		c.Trash++
	}
}

func inconclusiveReason(c Counts, required int) string {
	switch {
	case c.BootFailed > 0:
		return fmt.Sprintf("kernel does not boot (%d/%d attempts); fix the kernel first",
			c.BootFailed, c.Total())
	case c.Trash > 0 && c.Completed == 0:
		return fmt.Sprintf("no usable runs: %d/%d attempts failed on infrastructure", c.Trash, c.Total())
	case c.Trash > 0:
		return fmt.Sprintf("only %d/%d clean runs completed, %d attempts were unusable",
			c.Completed, required, c.Trash)
	default:
		return fmt.Sprintf("only %d/%d clean runs completed, check was interrupted",
			c.Completed, required)
	}
}

func (c Counts) Decide(required int) Result {
	if required <= 0 {
		return Result{
			Kind:     Inconclusive,
			Reason:   "required runs must be positive",
			Required: required,
		}
	}

	if c.Target > 0 {
		return Result{
			Kind: Reproduced,
			Reason: fmt.Sprintf(
				"target crash observed on run %d of %d (completed: %d, other crashes: %d, trash: %d)",
				c.Total(), c.Total(), c.Completed, c.Other, c.Trash,
			),
			Required: required,
		}
	}

	if c.Completed >= required {
		return Result{
			Kind: NotReproduced,
			Reason: fmt.Sprintf(
				"%d/%d runs completed with no target crash",
				c.Completed, required,
			),
			Required: required,
		}
	}

	return Result{
		Kind:     Inconclusive,
		Reason:   inconclusiveReason(c, required),
		Required: required,
	}
}
