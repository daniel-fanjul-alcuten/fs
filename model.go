package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Model map[string]Device

type Device struct {
	Source   string
	Target   string
	Commands Commands
	Authfile *string
	Local    *bool
}

type Commands struct {
	Check  string
	Mount  string
	Umount string
}

func (d Device) Run(command string, auth bool) (ok error, err error) {
	var f *os.File
	if auth && d.Authfile != nil {
		if f, err = ioutil.TempFile("", "fs-"); err != nil {
			return
		}
		defer os.Remove(f.Name())
		defer f.Close()
		w := bufio.NewWriter(f)
		authfile := *d.Authfile
		if _, err = w.WriteString(authfile); err != nil {
			return
		}
		if err = w.Flush(); err != nil {
			return
		}
		if err = f.Close(); err != nil {
			return
		}
	}
	cmd := exec.Command("bash", "-c", command)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "source="+d.Source)
	cmd.Env = append(cmd.Env, "target="+d.Target)
	if f != nil {
		cmd.Env = append(cmd.Env, "authfile="+f.Name())
	}
	PrintCommandString(command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ok = cmd.Run()
	if f != nil {
		if err = os.Remove(f.Name()); err != nil {
			return
		}
	}
	return
}

func Parse(fn string) (m Model, err error) {
	var reader io.Reader
	if strings.HasSuffix(fn, ".gpg") {
		output := &bytes.Buffer{}
		cmd := exec.Command("gpg", "--batch", "-d", fn)
		PrintCommand(cmd)
		cmd.Stdin = os.Stdin
		cmd.Stdout = output
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return
		}
		reader = output
	} else {
		var f *os.File
		if f, err = os.Open(fn); err != nil {
			return
		}
		defer f.Close()
		reader = bufio.NewReader(f)
	}
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(&m)
	return
}

func (m Model) Deps() (direct, reverse map[string][]string) {
	direct = make(map[string][]string)
	reverse = make(map[string][]string)
	for n1, d1 := range m {
		for n2, d2 := range m {
			if n1 != n2 && strings.HasPrefix(d1.Source, d2.Target) {
				direct[n1] = append(direct[n1], n2)
				reverse[n2] = append(reverse[n2], n1)
			}
		}
	}
	return
}

func (m Model) Locals() (locals map[string]*bool) {
	locals = map[string]*bool{}
	direct, _ := m.Deps()
	pending := map[string]struct{}{}
	for n := range m {
		pending[n] = struct{}{}
	}
	l := len(pending) + 1
	for l > len(pending) {
		l = len(pending)
		for n := range pending {
			d := m[n]
			if d.Local != nil {
				tmp := *d.Local
				locals[n] = &tmp
				delete(pending, n)
				continue
			}
			all, loc, rem := true, false, false
			for _, d := range direct[n] {
				ld := locals[d]
				all = all && ld != nil
				loc = loc || ld != nil && *ld
				rem = rem || ld != nil && !*ld
			}
			if rem {
				tmp := false
				locals[n] = &tmp
				delete(pending, n)
				continue
			}
			if all {
				if loc && !rem {
					tmp := true
					locals[n] = &tmp
				} else if !loc && rem {
					tmp := false
					locals[n] = &tmp
				} else {
					locals[n] = nil
				}
				delete(pending, n)
				continue
			}
		}
	}
	return
}

func (m Model) CheckFuncs() (funcs map[string]func() error) {
	funcs = map[string]func() error{}
	for n := range m {
		n, d := n, m[n]
		funcs[n] = func() (err error) {
			var ok error
			if ok, err = d.Run(d.Commands.Check, false); err != nil {
				return
			} else if ok == nil {
				PrintIsMounted(n)
			} else {
				PrintIsUmounted(n)
			}
			return
		}
	}
	return
}

func (m Model) DepFuncs(op string, r bool, f func(n string, d Device) error) (funcs map[string]func() error) {
	direct, reverse := m.Deps()
	if r {
		direct, reverse = reverse, nil
	}
	lock := sync.RWMutex{}
	cond := sync.NewCond(&lock)
	type state struct {
		work, done bool
		err        error
	}
	states := map[string]*state{}
	funcs = map[string]func() error{}
	for n := range m {
		n, d := n, m[n]
		states[n] = &state{}
		funcs[n] = func() (err error) {
			lock.Lock()
			for _, d := range direct[n] {
				go funcs[d]()
			}
			for _, d := range direct[n] {
				for !states[d].done {
					cond.Wait()
				}
				if err = states[d].err; err != nil {
					err = fmt.Errorf("%v %v: %v", op, n, err)
					states[n].work = true
					states[n].done = true
					states[n].err = err
					lock.Unlock()
					cond.Broadcast()
					return
				}
			}
			if !states[n].work {
				states[n].work = true
				go func() {
					err := f(n, d)
					lock.Lock()
					states[n].done = true
					states[n].err = err
					lock.Unlock()
					cond.Broadcast()
				}()
			}
			for !states[n].done {
				cond.Wait()
			}
			err = states[n].err
			lock.Unlock()
			return
		}
	}
	return
}

func (m Model) MountFuncs() (funcs map[string]func() error) {
	return m.DepFuncs("mount", false, func(n string, d Device) (err error) {
		var ok error
		if ok, err = d.Run(d.Commands.Check, false); err != nil {
			return
		} else if ok == nil {
			PrintIsMounted(n)
			return
		}
		if ok, err = d.Run(d.Commands.Mount, true); err != nil {
			return
		} else if ok == nil {
			PrintMounted(n)
			return
		}
		err = fmt.Errorf("mount %v: %v", n, ok)
		return
	})
}

func (m Model) UmountFuncs() (funcs map[string]func() error) {
	return m.DepFuncs("umount", true, func(n string, d Device) (err error) {
		var ok error
		if ok, err = d.Run(d.Commands.Check, false); err != nil {
			return
		} else if ok != nil {
			PrintIsUmounted(n)
			return
		}
		if ok, err = d.Run(d.Commands.Umount, false); err != nil {
			return
		} else if ok == nil {
			PrintUmounted(n)
			return
		}
		err = fmt.Errorf("umount %v: %v", n, ok)
		return
	})
}
