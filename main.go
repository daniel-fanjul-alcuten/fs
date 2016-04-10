package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	model, err := Parse(f)
	if err != nil {
		log.Fatal("fs: " + err.Error())
	}
	if bash != "" {
		ids := ""
		for n := range model {
			ids += " " + n
		}
		fmt.Printf(`%s() {
	local cur=${COMP_WORDS[COMP_CWORD]}
	if [ $COMP_CWORD -gt 1 ]; then
		local prev=${COMP_WORDS[$(($COMP_CWORD-1))]}
		case $prev in
			-f)
				COMPREPLY=()
				return
				;;
		esac
	fi
	case $cur in
		--*)
			local opts="--bash --f --l --r --a --m --u --v"
			COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
			return
			;;
	esac
	local opts="-bash -f -l -r -a -m -u -v`+ids+`"
	COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
}
`, bash)
		return
	}
	names := map[string]struct{}{}
	for _, n := range flag.Args() {
		if _, ok := model[n]; ok {
			names[n] = struct{}{}
		}
	}
	locals := model.Locals()
	if l {
		for n, l := range locals {
			if l != nil && *l {
				names[n] = struct{}{}
			}
		}
	}
	if r {
		for n, l := range locals {
			if l != nil && !*l {
				names[n] = struct{}{}
			}
		}
	}
	if g {
		for n, l := range locals {
			if l == nil {
				names[n] = struct{}{}
			}
		}
	}
	if flag.NArg() == 0 && !l && !r && !g {
		for n := range model {
			names[n] = struct{}{}
		}
	}
	var funcs map[string]func() error
	if m {
		funcs = model.MountFuncs()
	} else if u {
		funcs = model.UmountFuncs()
	} else {
		funcs = model.CheckFuncs()
	}
	errs := make(chan error, len(names))
	for n := range names {
		n := n
		go func() {
			errs <- funcs[n]()
		}()
	}
	var ferr error
	for i := 0; i < len(names); i++ {
		if err := <-errs; err != nil {
			log.Println("fs: " + err.Error())
			ferr = err
		}
	}
	if ferr != nil {
		os.Exit(1)
	}
}
