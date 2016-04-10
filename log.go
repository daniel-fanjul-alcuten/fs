package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

var (
	blue  string
	green string
	red   string
	reset string
)

func init() {
	cred := exec.Command("tput", "setaf", "1")
	bred := &bytes.Buffer{}
	cred.Stdout = bred
	if err := cred.Run(); err != nil {
		log.Fatal(err)
	}
	cgreen := exec.Command("tput", "setaf", "2")
	bgreen := &bytes.Buffer{}
	cgreen.Stdout = bgreen
	if err := cgreen.Run(); err != nil {
		log.Fatal(err)
	}
	cblue := exec.Command("tput", "setaf", "4")
	bblue := &bytes.Buffer{}
	cblue.Stdout = bblue
	if err := cblue.Run(); err != nil {
		log.Fatal(err)
	}
	creset := exec.Command("tput", "sgr0")
	breset := &bytes.Buffer{}
	creset.Stdout = breset
	if err := creset.Run(); err != nil {
		log.Fatal(err)
	}
	blue, green, red, reset =
		bblue.String(),
		bgreen.String(),
		bred.String(),
		breset.String()
	return
}

func PrintIsMounted(name string) {
	log.Printf("%s %smounted%s\n", name, green, reset)
}

func PrintIsUmounted(name string) {
	log.Printf("%s %sumounted%s\n", name, red, reset)
}

func PrintMounted(name string) {
	log.Printf("%s %smounted%s\n", name, blue, reset)
}

func PrintUmounted(name string) {
	log.Printf("%s %sumounted%s\n", name, blue, reset)
}

func PrintError(name string, err error) {
	log.Printf("%s: %s%s%s\n", name, red, err, reset)
}

func PrintCommand(cmd *exec.Cmd) {
	if v {
		fmt.Println("fs:", strings.Join(cmd.Args, " "))
	}
}

func PrintCommandString(cmd string) {
	if v {
		fmt.Println("fs:", cmd)
	}
}
