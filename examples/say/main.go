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

		// This argument reads environment variable and requires the variable must exist,
		// it doesn't accept input from command line.
		APIAccessKey string `cli:"#ER, The access key to your service provider" env:"MY_API_ACCESS_KEY"`
	}
	mcli.Parse(&args)
	fmt.Printf("Say to %s: %s\n", args.Name, args.Text)
}
