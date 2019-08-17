package main

// `cp` but dumb

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf("Usage: %s <infile> <outfile>\n", os.Args[0])
		os.Exit(1)
	}
	bs, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(flag.Arg(1), bs, 0777)
	if err != nil {
		panic(err)
	}
}
