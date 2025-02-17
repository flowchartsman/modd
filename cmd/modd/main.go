package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cortesi/termlog"
	"github.com/flowchartsman/modd"
	"github.com/flowchartsman/modd/notify"
	"gopkg.in/alecthomas/kingpin.v2"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

const modfile = "./modd.conf"

var file = kingpin.Flag(
	"file",
	fmt.Sprintf("Path to modfile (%s)", modfile),
).
	Default(modfile).
	PlaceHolder("PATH").
	Short('f').
	String()

var noconf = kingpin.Flag("noconf", "Don't watch our own config file").
	Short('c').
	Bool()

var beep = kingpin.Flag("bell", "Ring terminal bell if any command returns an error").
	Short('b').
	Bool()

var ignores = kingpin.Flag("ignores", "List default ignore patterns and exit").
	Short('i').
	Bool()

var doNotify = kingpin.Flag("notify", "Send stderr to system notification if commands error").
	Short('n').
	Bool()

var prep = kingpin.Flag("prep", "Run prep commands and exit").
	Short('p').
	Bool()

var debug = kingpin.Flag("debug", "Debugging for modd development").
	Default("false").
	Bool()

var exec = kingpin.Flag("exec", "Execute a command in the built-in shell").
	String()

var escapeExit = kingpin.Flag("escape-exit", "Will monitor the keyboard for the <ESC> key, and exit if pressed").
	Default("false").
	Bool()

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(modd.Version)
	kingpin.Parse()

	if *exec != "" {
		parser := syntax.NewParser()
		prog, err := parser.Parse(strings.NewReader(*exec), "")
		if err != nil {
			os.Exit(1)
		}

		runner, err := interp.New(
			interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
			func(r *interp.Runner) error {
				return nil
			},
		)
		if err != nil {
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		runner.Reset()
		err = runner.Run(ctx, prog)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *ignores {
		for _, patt := range modd.CommonExcludes {
			fmt.Println(patt)
		}
		os.Exit(0)
	}

	log := termlog.NewLog()
	if *debug {
		log.Enable("debug")
	}

	notifiers := []notify.Notifier{}
	if *doNotify {
		n := notify.PlatformNotifier()
		if n == nil {
			log.Shout("Could not find a desktop notifier")
		} else {
			notifiers = append(notifiers, n)
		}
	}
	if *beep {
		notifiers = append(notifiers, &notify.BeepNotifier{})
	}

	mr, err := modd.NewModRunner(*file, log, notifiers, !(*noconf), *escapeExit)
	if err != nil {
		log.Shout("%s", err)
		return
	}

	if *prep {
		err := mr.PrepOnly(true)
		if err != nil {
			log.Shout("%s", err)
		}
	} else {
		err = mr.Run()
		if err != nil {
			log.Shout("%s", err)
		}
	}
}
