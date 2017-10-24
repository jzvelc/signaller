package main

import (
	"github.com/codeskyblue/go-sh"
)

type Signal struct {
	Type    string
	Exec    []string `mapstructure:"exec"`
	Rewrite string   `mapstructure:"rewrite"`
	IsMute  bool     `mapstructure:"mute"`
}

type Unit struct {
	Name     string
	Exec     []string `mapstructure:"exec"`
	Callback []string `mapstructure:"callback"`

	IsMute    bool `mapstructure:"mute"`
	IsIsolate bool `mapstructure:"isolate"`

	IsRestart      bool `mapstructure:"restart"`
	RestartTimeout int  `mapstructure:"restart_timeout"`

	Session *sh.Session

	Env []map[string]string `mapstructure:"env"`

	Signals []map[string][]*Signal `mapstructure:"signal"`
}

func (u *Unit) GetEnv(env map[string]string) (e map[string]string) {
	e = make(map[string]string)

	if !u.IsIsolate {
		for k, v := range env {
			e[k] = v
		}
	}

	for _, s := range u.Env {
		for k, v := range s {
			e[k] = v
		}
	}

	return e
}

func (u *Unit) GetSignals() (signals []*Signal) {

	for _, s := range u.Signals {
		for k, v := range s {
			v[0].Type = k
			signals = append(signals, v[0])
		}
	}

	return signals
}

type Config struct {
	ExitSignal  string               `mapstructure:"exit_signal"`
	ExitTimeout int                  `mapstructure:"exit_timeout"`
	TermSignal  string               `mapstructure:"term_signal"`
	Units       []map[string][]*Unit `mapstructure:"unit"`
}

func (c *Config) GetUnits() (units []*Unit) {
	for _, s := range c.Units {
		for k, v := range s {
			v[0].Name = k
			units = append(units, v[0])
		}
	}

	return units
}
