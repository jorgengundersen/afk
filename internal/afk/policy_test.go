package afk

import "testing"

func TestDecideMainChildExit(t *testing.T) {
	tests := []struct {
		name         string
		fail         string
		untilSuccess bool
		exitCode     int
		want         mainChildExitDecision
	}{
		{
			name:     "default continue ignores zero for stopping",
			fail:     "continue",
			exitCode: 0,
			want:     mainChildExitDecision{},
		},
		{
			name:     "default continue ignores nonzero for stopping",
			fail:     "continue",
			exitCode: 7,
			want:     mainChildExitDecision{},
		},
		{
			name:     "fail stop continues after zero",
			fail:     "stop",
			exitCode: 0,
			want:     mainChildExitDecision{},
		},
		{
			name:     "fail stop stops with nonzero exit code",
			fail:     "stop",
			exitCode: 143,
			want:     mainChildExitDecision{stop: true, exitCode: 143},
		},
		{
			name:         "until success stops on zero",
			fail:         "continue",
			untilSuccess: true,
			exitCode:     0,
			want:         mainChildExitDecision{stop: true, exitCode: 0},
		},
		{
			name:         "until success records nonzero and continues",
			fail:         "stop",
			untilSuccess: true,
			exitCode:     124,
			want:         mainChildExitDecision{recordLastNonZero: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decideMainChildExit(tc.fail, tc.untilSuccess, tc.exitCode)
			if got != tc.want {
				t.Fatalf("decideMainChildExit() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestFinalExitAfterCompletedWork(t *testing.T) {
	tests := []struct {
		name         string
		untilSuccess bool
		lastNonZero  int
		want         int
	}{
		{name: "completed work under continue exits zero", want: 0},
		{name: "exhausted until-success exits last nonzero", untilSuccess: true, lastNonZero: 124, want: 124},
		{name: "exhausted until-success without recorded failure exits one", untilSuccess: true, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := finalExitAfterCompletedWork(tc.untilSuccess, tc.lastNonZero)
			if got != tc.want {
				t.Fatalf("finalExitAfterCompletedWork() = %d, want %d", got, tc.want)
			}
		})
	}
}
