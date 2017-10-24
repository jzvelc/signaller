package main

import (
	"github.com/codeskyblue/go-sh"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"github.com/hashicorp/hcl"
	"io/ioutil"
	"github.com/mitchellh/mapstructure"
	"io"
	"fmt"
	"time"
	"bytes"
	"bufio"
	"sync"
)

// Version will be filled in by the compiler.
var Version string

type PrefixWriter struct {
	dst    io.Writer
	prefix string
}

func NewPrefixWriter(dst io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{dst, prefix}
}

func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	t := pw.prefix + string(p)
	t = strings.Replace(t, "\n", "\n"+pw.prefix, strings.Count(t, "\n")-1)
	_, err = pw.dst.Write([]byte(t))
	return len(p), err
}

func main() {
	log.Println("[INFO] running signaller", Version)

	if len(os.Args) < 2 {
		panic("Missing config file")
	}

	configFile := os.Args[1]

	var config *Config
	{
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}

		obj, err := hcl.Parse(string(b))
		if err != nil {
			panic(err)
		}

		c := make(map[string]interface{})
		if err = hcl.DecodeObject(&c, obj); err != nil {
			panic(err)
		}

		dc := &mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           &config,
			WeaklyTypedInput: true,
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		}

		decoder, err := mapstructure.NewDecoder(dc)
		if err != nil {
			panic(err)
		}

		err = decoder.Decode(c)
		if err != nil {
			panic(err)
		}
	}

	if config.ExitTimeout == 0 {
		config.ExitTimeout = 5
	}

	// Create signal map
	signals := make(map[string]os.Signal)
	signals["ABRT"] = syscall.SIGABRT
	signals["ALRM"] = syscall.SIGALRM
	signals["BUS"] = syscall.SIGBUS
	signals["CHLD"] = syscall.SIGCHLD
	signals["CLD"] = syscall.SIGCLD
	signals["CONT"] = syscall.SIGCONT
	signals["FPE"] = syscall.SIGFPE
	signals["HUP"] = syscall.SIGHUP
	signals["ILL"] = syscall.SIGILL
	signals["INT"] = syscall.SIGINT
	signals["IO"] = syscall.SIGIO
	signals["IOT"] = syscall.SIGIOT
	signals["KILL"] = syscall.SIGKILL
	signals["PIPE"] = syscall.SIGPIPE
	signals["POLL"] = syscall.SIGPOLL
	signals["PROF"] = syscall.SIGPROF
	signals["PWR"] = syscall.SIGPWR
	signals["QUIT"] = syscall.SIGQUIT
	signals["SEGV"] = syscall.SIGSEGV
	signals["STKFLT"] = syscall.SIGSTKFLT
	signals["STOP"] = syscall.SIGSTOP
	signals["SYS"] = syscall.SIGSYS
	signals["TERM"] = syscall.SIGTERM
	signals["TRAP"] = syscall.SIGTRAP
	signals["TSTP"] = syscall.SIGTSTP
	signals["TTIN"] = syscall.SIGTTIN
	signals["TTOU"] = syscall.SIGTTOU
	signals["UNUSED"] = syscall.SIGUNUSED
	signals["URG"] = syscall.SIGURG
	signals["USR1"] = syscall.SIGUSR1
	signals["USR2"] = syscall.SIGUSR2
	signals["VTALRM"] = syscall.SIGVTALRM
	signals["WINCH"] = syscall.SIGWINCH
	signals["XCPU"] = syscall.SIGXCPU
	signals["XFSZ"] = syscall.SIGXFSZ

	env := make(map[string]string)
	for _, e := range os.Environ() {
		v := strings.Split(e, "=")
		env[v[0]] = v[1]
	}

	// Get units
	units := config.GetUnits()

	var mutex = &sync.Mutex{}
	var ag ActorGroup
	interrupt := make(chan struct{}, len(units)+1)
	ag.Add(func() error {
		for {
			var l []os.Signal
			for _, s := range signals {
				l = append(l, s)
			}

			c := make(chan os.Signal, 1)

			signal.Notify(c, l...)
			select {
			case sig := <-c:
				signalName := ""
				for k, s := range signals {
					if s == sig {
						signalName = k
						break
					}
				}

				log.Println("[DEBUG] caught signal:", signalName)

				for _, u := range units {
					unitSignals := u.GetSignals()

					if len(unitSignals) == 0 && u.Session != nil {
						log.Println("[INFO] signalling:", u.Name, "-->", signalName)
						u.Session.Kill(sig)
						continue
					}

					for _, s := range unitSignals {
						names := strings.Split(s.Type, "|")
						isExist := false

						for _, n := range names {
							if n == "*" || n == signalName {
								isExist = true
								break
							}
						}

						if isExist {
							isError := false
							if len(s.Exec) > 0 {
								log.Println("[INFO] executing command:", u.Name, "-->", strings.Join(s.Exec, " "))
								session := createSession(u, s.IsMute, env)
								err := session.Command(s.Exec[0], s.Exec[1:]).Run()
								if err != nil {
									isError = true
									log.Println("[ERROR] failed executing command:", u.Name, "-->", err)
								}
							}

							if !isError && u.Session != nil {
								if s.Rewrite != "" {
									if s.Rewrite != "" {
										log.Println("[INFO] signalling:", u.Name, "-->", s.Rewrite)
										u.Session.Kill(signals[s.Rewrite])
									}
								} else {
									log.Println("[INFO] signalling:", u.Name, "-->", signalName)
									u.Session.Kill(sig)
								}
							}
						}
					}
				}

				// Terminate
				if signals[config.TermSignal] == sig {
					mutex.Lock()

					log.Println("[INFO] terminating...")
					for _, u := range units {
						if u.Session != nil {
							u.Session.Kill(syscall.SIGKILL)
						}
					}

					os.Exit(0)

					return nil
				}

				if signals["INT"] == sig || signals[config.ExitSignal] == sig {
					return nil
				}
			case <-interrupt:
				return nil
			}
		}
	}, func(err error) {
		mutex.Lock()

		// Graceful exit
		log.Println("[INFO] exiting...")
		time.Sleep(time.Second)
		for _, u := range units {
			u.IsRestart = false
			if u.Session != nil {
				u.Session.Kill(syscall.SIGQUIT)
			}
		}
		interrupt <- struct{}{}

		// Terminate after exit timeout
		go func() {
			time.Sleep(time.Second * time.Duration(config.ExitTimeout))
			log.Println("[INFO] terminating...")
			for _, u := range units {
				if u.Session != nil {
					u.Session.Kill(syscall.SIGKILL)
				}
			}
			os.Exit(0)
		}()
	})

	minSatisfy := 0
	for _, u := range units {
		if len(u.Exec) == 0 {
			continue
		}

		minSatisfy += 1

		func(u *Unit) {
			ag.Add(func() error {
				for {
					mutex.Lock()

					u.Session = createSession(u, u.IsMute, env)
					u.Session.Command(u.Exec[0], u.Exec[1:])

					mutex.Unlock()

					err := u.Session.Run()

					if err != nil && len(u.Callback) > 0 {
						log.Println("[INFO] executing callback:", u.Name, "-->", strings.Join(u.Callback, " "))
						session := createSession(u, u.IsMute, env)
						session.Command(u.Callback[0], u.Callback[1:]).Run()
					}

					restartTimeout := u.RestartTimeout
					if restartTimeout == 0 {
						restartTimeout = 5
					}

					if err != nil && u.IsRestart {
						log.Println("[ERROR] unit", u.Name, "failed:", err.Error())

						select {
						case <-interrupt:
							return err
						case <-time.After(time.Second * time.Duration(restartTimeout)):
							break
						}

						log.Println("[INFO] restarting", u.Name)

						continue
					}

					return err
				}
			}, func(err error) {
				if err != nil {
					for _, u := range units {
						u.IsRestart = false
					}

					interrupt <- struct{}{}
				}
			})
		}(u)
	}

	err := ag.RunUntilError(minSatisfy)
	if err != nil {
		panic(err)
	}
}

func createSession(u *Unit, isMute bool, env map[string]string) (session *sh.Session) {
	session = sh.NewSession()
	session.Env = u.GetEnv(env)

	if isMute {
		var b bytes.Buffer
		buf := bufio.NewWriter(&b)
		session.Stdout = buf
		session.Stderr = buf
	} else {
		pw := NewPrefixWriter(os.Stdout, fmt.Sprintf("[%s] ", u.Name))
		session.Stdout = pw
		session.Stderr = pw
	}

	return session
}

