package afk

import (
	"reflect"
	"testing"
)

func TestParseArgs_SplitsAtFirstSeparatorAndPreservesChildArgv(t *testing.T) {
	cfg, err := ParseArgs([]string{"--loops", "2", "--", "echo", "--", "literal", "value"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []string{"echo", "--", "literal", "value"}
	if !reflect.DeepEqual(cfg.CommandArgv, want) {
		t.Fatalf("CommandArgv = %#v, want %#v", cfg.CommandArgv, want)
	}
}

func TestParseArgs_HelpWithoutSeparator(t *testing.T) {
	cfg, err := ParseArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if !cfg.Help {
		t.Fatalf("Help = %v, want true", cfg.Help)
	}
}

func TestParseArgs_TracksExplicitFlagsUsedForValidation(t *testing.T) {
	defaults, err := ParseArgs([]string{"--loops", "1", "--", "echo", "ok"})
	if err != nil {
		t.Fatalf("ParseArgs(defaults) error = %v", err)
	}
	if defaults.SleepExplicit || defaults.EmptySleepsExplicit || defaults.FailExplicit {
		t.Fatalf("defaults explicit tracking = sleep:%v empty:%v fail:%v, want all false", defaults.SleepExplicit, defaults.EmptySleepsExplicit, defaults.FailExplicit)
	}

	explicit, err := ParseArgs([]string{"--loops", "1", "--sleep", "5s", "--empty-sleeps", "2", "--fail", "stop", "--", "echo"})
	if err != nil {
		t.Fatalf("ParseArgs(explicit) error = %v", err)
	}
	if !explicit.SleepExplicit || !explicit.EmptySleepsExplicit || !explicit.FailExplicit {
		t.Fatalf("explicit tracking = sleep:%v empty:%v fail:%v, want all true", explicit.SleepExplicit, explicit.EmptySleepsExplicit, explicit.FailExplicit)
	}
}
