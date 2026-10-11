package main

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func TestOptions(t *testing.T) {
	for _, tc := range []struct {
		name              string
		args              []string
		noise, device     string
		volume, low, high float64
	}{
		{"defaults", nil, "pink", "all", .2, 300, 3000},
		{"long", []string{"pink", "--volume", "0.3", "--freq", "400-4000", "--device", "current"}, "pink", "current", .3, 400, 4000},
		{"short", []string{"w", "-v", "0.1", "-f", "100~5000", "-d", "current"}, "white", "current", .1, 100, 5000},
		{"white", []string{"white"}, "white", "all", .2, 20, 20000},
		{"equals", []string{"p", "-v=0", "--freq=200-2000"}, "pink", "all", 0, 200, 2000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := parseOptions(tc.args, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if o.noise != tc.noise || o.device != tc.device || o.volume != tc.volume || o.low != tc.low || o.high != tc.high {
				t.Fatalf("unexpected options: %+v", o)
			}
		})
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"brown"}, {"--noise", "pink"}, {"-volume", ".2"}, {"--v", ".2"}, {"-xxx"},
		{"-v", "-1"}, {"-v", "1.1"}, {"-v", "NaN"}, {"-v", "Inf"}, {"-v"},
		{"-f", "3000-300"}, {"-f", "0-3000"}, {"-f", "300-22050"}, {"-f", "NaN-3000"},
		{"-f", "100"}, {"-f", "100-Inf"}, {"-f", "100-100"}, {"-d", "other"},
		{"-t", "-1s"}, {"pink", "white"}, {"-v", ".2", "white"},
	} {
		if _, err := parseOptions(args, io.Discard); err == nil {
			t.Errorf("accepted invalid args %v", args)
		}
	}
}

func TestHelpAndUtilityOptions(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		if _, err := parseOptions([]string{arg}, io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("%s: %v", arg, err)
		}
	}
	for _, args := range [][]string{{"-l", "-t", "1s"}, {"--list-devices", "--duration=1s"}} {
		o, err := parseOptions(args, io.Discard)
		if err != nil || !o.list || o.duration.Seconds() != 1 {
			t.Fatalf("%v: %+v, %v", args, o, err)
		}
	}
}
