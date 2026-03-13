package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jorgengundersen/afk/config"
)

func validConfig() config.Config {
	return config.Config{
		Mode:    config.MaxIterationsMode,
		Harness: "claude",
		Prompt:  "do the thing",
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name: "valid_max_iterations_with_prompt",
			cfg:  validConfig(),
		},
		{
			name: "valid_daemon_with_beads",
			cfg: config.Config{
				Mode:         config.DaemonMode,
				Harness:      "claude",
				BeadsEnabled: true,
			},
		},
		{
			name: "valid_raw_command_with_prompt",
			cfg: config.Config{
				Mode:       config.MaxIterationsMode,
				RawCommand: "aider --yes {prompt}",
				Prompt:     "fix it",
			},
		},
		{
			name: "raw_command_with_harness",
			cfg: config.Config{
				Mode:       config.MaxIterationsMode,
				RawCommand: "aider {prompt}",
				Harness:    "claude",
				Prompt:     "fix it",
			},
			wantErr: "--raw cannot be combined with --harness, --model, or --agent-flags",
		},
		{
			name: "raw_command_with_model",
			cfg: config.Config{
				Mode:       config.MaxIterationsMode,
				RawCommand: "aider {prompt}",
				Model:      "gpt-4o",
				Prompt:     "fix it",
			},
			wantErr: "--raw cannot be combined with --harness, --model, or --agent-flags",
		},
		{
			name: "raw_command_with_agent_flags",
			cfg: config.Config{
				Mode:       config.MaxIterationsMode,
				RawCommand: "aider {prompt}",
				AgentFlags: "--verbose",
				Prompt:     "fix it",
			},
			wantErr: "--raw cannot be combined with --harness, --model, or --agent-flags",
		},
		{
			name: "quiet_with_stderr",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "claude",
				Prompt:  "fix it",
				Quiet:   true,
				Stderr:  true,
			},
			wantErr: "--quiet cannot be combined with --stderr or --verbose",
		},
		{
			name: "quiet_with_verbose",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "claude",
				Prompt:  "fix it",
				Quiet:   true,
				Verbose: true,
			},
			wantErr: "--quiet cannot be combined with --stderr or --verbose",
		},
		{
			name: "no_prompt_no_beads",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "claude",
			},
			wantErr: "no prompt provided and beads integration is not active; nothing to do",
		},
		{
			name: "sleep_without_daemon",
			cfg: config.Config{
				Mode:          config.MaxIterationsMode,
				Harness:       "claude",
				Prompt:        "fix it",
				SleepInterval: 60 * time.Second,
			},
			wantErr: "--sleep requires --daemon mode",
		},
		{
			name: "unknown_harness",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "unknown-agent",
				Prompt:  "fix it",
			},
			wantErr: `unknown harness "unknown-agent"`,
		},
		{
			name: "empty_harness_no_raw",
			cfg: config.Config{
				Mode:   config.MaxIterationsMode,
				Prompt: "fix it",
			},
			wantErr: "harness is required when --raw is not set",
		},
		{
			name: "valid_all_known_harnesses_claude",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "claude",
				Prompt:  "fix it",
			},
		},
		{
			name: "valid_all_known_harnesses_opencode",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "opencode",
				Prompt:  "fix it",
			},
		},
		{
			name: "valid_all_known_harnesses_codex",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "codex",
				Prompt:  "fix it",
			},
		},
		{
			name: "valid_all_known_harnesses_copilot",
			cfg: config.Config{
				Mode:    config.MaxIterationsMode,
				Harness: "copilot",
				Prompt:  "fix it",
			},
		},
		{
			name: "valid_daemon_with_sleep",
			cfg: config.Config{
				Mode:          config.DaemonMode,
				Harness:       "claude",
				BeadsEnabled:  true,
				SleepInterval: 120 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() returned nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
