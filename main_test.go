package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_Dispatch(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
		wantStderr []string // substrings stderr must contain; empty = stderr must be empty
	}{
		{
			name:       "version flag",
			args:       []string{"--version"},
			wantExit:   0,
			wantStdout: "zt-sim version 0.1.0\n",
		},
		{
			name:       "no arguments",
			args:       nil,
			wantExit:   64,
			wantStderr: []string{"usage: zt-sim <command>"},
		},
		{
			name:       "unknown subcommand",
			args:       []string{"frobnicate"},
			wantExit:   64,
			wantStderr: []string{`unknown subcommand "frobnicate"`, "usage: zt-sim <command>"},
		},
		{
			name:       "decide is stubbed",
			args:       []string{"decide"},
			wantExit:   70,
			wantStderr: []string{"decide: not implemented yet"},
		},
		{
			name:       "run is stubbed",
			args:       []string{"run"},
			wantExit:   70,
			wantStderr: []string{"run: not implemented yet"},
		},
		{
			name:       "policies is stubbed",
			args:       []string{"policies"},
			wantExit:   70,
			wantStderr: []string{"policies: not implemented yet"},
		},
		{
			name:       "audit is stubbed",
			args:       []string{"audit"},
			wantExit:   70,
			wantStderr: []string{"audit: not implemented yet"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run(tc.args, &stdout, &stderr)

			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d", exit, tc.wantExit)
			}
			if got := stdout.String(); got != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", got, tc.wantStdout)
			}
			gotErr := stderr.String()
			if len(tc.wantStderr) == 0 {
				if gotErr != "" {
					t.Errorf("stderr = %q, want empty", gotErr)
				}
				return
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(gotErr, want) {
					t.Errorf("stderr %q does not contain %q", gotErr, want)
				}
			}
		})
	}
}
