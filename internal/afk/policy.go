package afk

type mainChildExitDecision struct {
	stop              bool
	exitCode          int
	recordLastNonZero bool
}

func decideMainChildExit(fail string, untilSuccess bool, exitCode int) mainChildExitDecision {
	if untilSuccess {
		if exitCode == 0 {
			return mainChildExitDecision{stop: true, exitCode: 0}
		}
		return mainChildExitDecision{recordLastNonZero: true}
	}

	if fail == "stop" && exitCode != 0 {
		return mainChildExitDecision{stop: true, exitCode: exitCode}
	}

	return mainChildExitDecision{}
}

func finalExitAfterCompletedWork(untilSuccess bool, lastNonZero int) int {
	if untilSuccess {
		if lastNonZero != 0 {
			return lastNonZero
		}
		return 1
	}
	return 0
}
