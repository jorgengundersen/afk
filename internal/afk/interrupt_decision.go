package afk

import "os"

type interruptShutdownPhase int

const (
	interruptPhaseWaitingForFirstSigint interruptShutdownPhase = iota
	interruptPhaseGracefulShutdown
)

type interruptDecision int

const (
	interruptDecisionIgnore interruptDecision = iota
	interruptDecisionStartGracefulShutdown
	interruptDecisionForceHardShutdown
)

type interruptShutdownState struct {
	phase interruptShutdownPhase
}

func newInterruptShutdownState() interruptShutdownState {
	return interruptShutdownState{phase: interruptPhaseWaitingForFirstSigint}
}

func (s *interruptShutdownState) observe(sig os.Signal) interruptDecision {
	decision := decideInterrupt(sig, s.phase)
	if decision == interruptDecisionStartGracefulShutdown {
		s.phase = interruptPhaseGracefulShutdown
	}
	return decision
}

func decideInterrupt(sig os.Signal, phase interruptShutdownPhase) interruptDecision {
	if !isSigint(sig) {
		return interruptDecisionIgnore
	}
	if phase == interruptPhaseGracefulShutdown {
		return interruptDecisionForceHardShutdown
	}
	return interruptDecisionStartGracefulShutdown
}
