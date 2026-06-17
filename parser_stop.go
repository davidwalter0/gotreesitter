package gotreesitter

import (
	"sync/atomic"
	"time"
)

type parserStopBudgetState struct {
	active   bool
	deadline time.Time
}

func (p *Parser) beginParseStopBudget(start time.Time) parserStopBudgetState {
	if p == nil {
		return parserStopBudgetState{}
	}
	prev := parserStopBudgetState{
		active:   p.parseStopBudgetActive,
		deadline: p.parseStopDeadline,
	}
	if p.parseStopBudgetActive {
		return prev
	}
	p.parseStopBudgetActive = true
	if p.timeoutMicros > 0 {
		p.parseStopDeadline = start.Add(time.Duration(p.timeoutMicros) * time.Microsecond)
	} else {
		p.parseStopDeadline = time.Time{}
	}
	return prev
}

func (p *Parser) restoreParseStopBudget(prev parserStopBudgetState) {
	if p == nil {
		return
	}
	p.parseStopBudgetActive = prev.active
	p.parseStopDeadline = prev.deadline
}

func (p *Parser) beginParseOperationBudget() func() {
	prev := p.beginParseStopBudget(time.Now())
	return func() {
		p.restoreParseStopBudget(prev)
	}
}

func (p *Parser) parseStopReasonNow() ParseStopReason {
	if p == nil {
		return ParseStopNone
	}
	if p.parseStopBudgetActive && !p.parseStopDeadline.IsZero() && time.Now().After(p.parseStopDeadline) {
		return ParseStopTimeout
	}
	if flag := p.cancellationFlag; flag != nil && atomic.LoadUint32(flag) != 0 {
		return ParseStopCancelled
	}
	return ParseStopNone
}

func parseStopReasonIsTerminal(reason ParseStopReason) bool {
	return reason == ParseStopTimeout || reason == ParseStopCancelled
}
