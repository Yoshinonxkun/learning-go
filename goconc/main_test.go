package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []cmdArgs
	}{
		{
			"zwei Befehle",
			[]string{"abc", "def", "::", "abcd", "defg"},
			[]cmdArgs{
				{"abc", []string{"def"}},
				{"abcd", []string{"defg"}},
			},
		},
		{
			"ein Befehl",
			[]string{"abc", "def", "hij"},
			[]cmdArgs{
				{"abc", []string{"def", "hij"}},
			},
		},
		{
			"kein Befehl",
			[]string{},
			nil,
		},
		{
			"zwei Befehle kurz deklariert",
			[]string{"abc", "def", ":", "defg"},
			[]cmdArgs{
				{"abc", []string{"def"}},
				{"abc", []string{"defg"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
