package main

import (
	"fmt"

	"github.com/jxskiss/mcli"
)

// say -n Daniel hello
// say hello -n Daniel

func main() {
	var args struct {
		Name string `cli:"-n, --name, Who do you want to say to" default:"tom"`

		// This argument is required.
		Text string `cli:"#R, text, The 'message' you want to send"`
	}
	fs, _ := mcli.Parse(&args)

	fmt.Printf("Say to %s: %s\n", args.Name, args.Text)
	fmt.Printf("fs.Args: %v\n", fs.Args())
}
