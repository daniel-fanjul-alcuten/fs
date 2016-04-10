package main

import (
	"flag"
	"log"
	"os/user"
	"path/filepath"
)

var (
	bash    string
	f       string
	l, r, g bool
	m, u    bool
	v       bool
)

func init() {
	me, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	fn := filepath.Join(me.HomeDir, ".fs.json.gpg")
	flag.StringVar(&f, "f", fn, "config file name")
	flag.BoolVar(&l, "l", false, "local")
	flag.BoolVar(&r, "r", false, "remote")
	flag.BoolVar(&g, "?", false, "unknown")
	flag.BoolVar(&m, "m", false, "mount")
	flag.BoolVar(&u, "u", false, "umount")
	flag.BoolVar(&v, "v", false, "verbose")
	flag.StringVar(&bash, "bash", "", "bash function for completion")
}
