package afk

import (
	"reflect"
	"testing"
	"time"
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
	defaults, err := ParseArgs([]string{"--daemon", "--", "echo", "ok"})
	if err != nil {
		t.Fatalf("ParseArgs(defaults) error = %v", err)
	}
	if defaults.SleepExplicit || defaults.EmptySleepsExplicit || defaults.FailExplicit || defaults.LoopsExplicit || defaults.ItemsExplicit || defaults.ItemsCmdExplicit || defaults.TimeoutExplicit {
		t.Fatalf("defaults explicit tracking = sleep:%v empty:%v fail:%v loops:%v items:%v itemsCmd:%v timeout:%v, want all false", defaults.SleepExplicit, defaults.EmptySleepsExplicit, defaults.FailExplicit, defaults.LoopsExplicit, defaults.ItemsExplicit, defaults.ItemsCmdExplicit, defaults.TimeoutExplicit)
	}

	explicit, err := ParseArgs([]string{"--loops", "1", "--sleep", "5s", "--empty-sleeps", "2", "--fail", "stop", "--timeout", "1s", "--", "echo"})
	if err != nil {
		t.Fatalf("ParseArgs(explicit) error = %v", err)
	}
	if !explicit.SleepExplicit || !explicit.EmptySleepsExplicit || !explicit.FailExplicit || !explicit.LoopsExplicit || explicit.ItemsExplicit || explicit.ItemsCmdExplicit || !explicit.TimeoutExplicit {
		t.Fatalf("explicit tracking = sleep:%v empty:%v fail:%v loops:%v items:%v itemsCmd:%v timeout:%v", explicit.SleepExplicit, explicit.EmptySleepsExplicit, explicit.FailExplicit, explicit.LoopsExplicit, explicit.ItemsExplicit, explicit.ItemsCmdExplicit, explicit.TimeoutExplicit)
	}

	emptyItems, err := ParseArgs([]string{"--items", "", "--", "echo"})
	if err != nil {
		t.Fatalf("ParseArgs(emptyItems) error = %v", err)
	}
	if !emptyItems.ItemsExplicit {
		t.Fatalf("ItemsExplicit = %v, want true", emptyItems.ItemsExplicit)
	}
}

func TestValidateConfig_UsageRules(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		ok   bool
	}{
		{
			name: "valid loops",
			cfg:  Config{Loops: 1, LoopsExplicit: true, CommandArgv: []string{"echo"}},
			ok:   true,
		},
		{
			name: "valid daemon with items-cmd and explicit sleep",
			cfg:  Config{Daemon: true, ItemsCmd: "./list", ItemsCmdExplicit: true, Sleep: time.Second, SleepExplicit: true, CommandArgv: []string{"echo"}},
			ok:   true,
		},
		{
			name: "missing loop driver",
			cfg:  Config{CommandArgv: []string{"echo"}},
		},
		{
			name: "missing main child command",
			cfg:  Config{Loops: 1, LoopsExplicit: true},
		},
		{
			name: "loops must be positive when explicit",
			cfg:  Config{Loops: 0, LoopsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "loops and daemon are mutually exclusive",
			cfg:  Config{Loops: 1, LoopsExplicit: true, Daemon: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "loops and items are mutually exclusive",
			cfg:  Config{Loops: 1, LoopsExplicit: true, Items: "x", ItemsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "loops and items-cmd are mutually exclusive",
			cfg:  Config{Loops: 1, LoopsExplicit: true, ItemsCmd: "x", ItemsCmdExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "items and items-cmd are mutually exclusive",
			cfg:  Config{Items: "x", ItemsExplicit: true, ItemsCmd: "x", ItemsCmdExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "daemon and items are mutually exclusive",
			cfg:  Config{Daemon: true, Items: "x", ItemsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "explicit sleep requires items-cmd",
			cfg:  Config{Loops: 1, LoopsExplicit: true, Sleep: time.Second, SleepExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "explicit sleep must be non-negative",
			cfg:  Config{ItemsCmd: "x", ItemsCmdExplicit: true, Sleep: -time.Second, SleepExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "explicit empty-sleeps requires items-cmd",
			cfg:  Config{Loops: 1, LoopsExplicit: true, EmptySleeps: 1, EmptySleepsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "explicit empty-sleeps must be non-negative",
			cfg:  Config{ItemsCmd: "x", ItemsCmdExplicit: true, EmptySleeps: -1, EmptySleepsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "fail value must be continue or stop",
			cfg:  Config{Loops: 1, LoopsExplicit: true, Fail: "bad", FailExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "until-success invalid with explicit fail",
			cfg:  Config{Loops: 1, LoopsExplicit: true, UntilSuccess: true, Fail: "stop", FailExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "until-success requires loops or daemon",
			cfg:  Config{UntilSuccess: true, ItemsCmd: "x", ItemsCmdExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "until-success invalid with items",
			cfg:  Config{UntilSuccess: true, Loops: 1, LoopsExplicit: true, Items: "x", ItemsExplicit: true, CommandArgv: []string{"echo"}},
		},
		{
			name: "timeout when explicit must be positive",
			cfg:  Config{Loops: 1, LoopsExplicit: true, TimeoutExplicit: true, Timeout: 0, CommandArgv: []string{"echo"}},
		},
		{
			name: "timeout when set negative is invalid",
			cfg:  Config{Loops: 1, LoopsExplicit: true, Timeout: -time.Second, CommandArgv: []string{"echo"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(tc.cfg)
			if tc.ok {
				if err != nil {
					t.Fatalf("ValidateConfig() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("ValidateConfig() error = nil, want non-nil")
			}
		})
	}
}
