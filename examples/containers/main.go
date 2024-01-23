package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/jxskiss/mcli"
)

/*
NAME:
   con - Running container handler written in go with tab completion on root command

USAGE:
   con [containers...] [global options] [root command options]

VERSION:
   dev

COMMANDS:
   root/no name    Stop and eventually remove running container

OPTIONS:
   containers      Container ids to stop.
   --rm            Trigger removal of container after stop.

GLOBAL OPTIONS:
   --dry-run, -r   Add a domain to the process. Can be specified multiple times.
   --engine value  Container engine on which to execute commands. (default: "docker") [$CON_ENGINE]
*/

type GlobalFlags struct {
	Engine string `cli:"--engine, Container engine to run command on." env:"CON_ENGINE" default:"docker"`
	DryRun bool   `cli:"-r, --dry-run, Show commands without execution"`
}

var globalFlags GlobalFlags

func main() {
	app := &mcli.App{
		Description: `Stop and remove running containers`,
	}
	app.SetGlobalFlags(&globalFlags)
	app.AddRoot(cmdRoot)
	app.AddCompletion()
	app.AddHelp()
	app.Options.EnableFlagCompletionForAllCommands = true
	app.Run()
}

func cmdRoot(ctx *mcli.Context) {
	var args struct {
		Containers []string `cli:"containers"`
		Rm         bool     `cli:"-x, --rm, Remove container after stop."`
	}
	funcs := make(map[string]mcli.ArgCompletionFunc)
	funcs["containers"] = CompleteContainers
	ctx.Parse(&args, mcli.WithArgCompFuncs(funcs))

	bin := globalFlags.Engine
	dry := globalFlags.DryRun

	if dry {
		log.Println("Dry run")
	}

	if !dry {
		_, err := exec.LookPath(bin)
		if err != nil {
			log.Printf("Cannot find executable %s\n", bin)
			os.Exit(1)
		}
	}
	log.Printf("Using engine %s\n", globalFlags.Engine)

	for _, c := range args.Containers {
		log.Printf("Stopping container with stop %s\n", c)

		if !dry {
			err := exec.Command(bin, "stop", c).Run()
			if err != nil {
				log.Printf("Stopping failed %+v\n", err)
				os.Exit(1)
			}
		}
		log.Printf("Stopped container %s\n", c)

		if args.Rm {
			log.Printf("Removing container with rm %s\n", c)
			if !dry {
				err := exec.Command(bin, "rm", c).Run()
				if err != nil {
					log.Printf("Stopping failed %+v\n", err)
					os.Exit(1)
				}
			}
			log.Printf("Removed container %s\n", c)
		}
	}
}

func CompleteContainers(_ mcli.ArgCompletionContext) []mcli.CompletionItem {
	options := []mcli.CompletionItem{}
	args := os.Args[1:]

	bin := "docker"
	if engine, ok := os.LookupEnv("CON_ENGINE"); ok {
		bin = engine
	}

	_, err := exec.LookPath(bin)
	if err != nil {
		log.Printf("Cannot find executable %s\n", bin)
		os.Exit(1)
	}

	params := []string{"ps", "--format", "{{.ID}};{{.Names}};{{.Image}}"}
	out, err := exec.Command(bin, params...).Output()
	if err != nil {
		log.Printf("Command execution failed %+v\n", err)
		os.Exit(1)
	}

	r := csv.NewReader(bytes.NewReader(out))
	r.Comma = ';'
	r.Comment = '#'

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if len(record[0]) > 20 || Contains(args, record[0]) {
			continue
		}

		options = append(
			options,
			mcli.CompletionItem{
				Value:       record[0],
				Description: fmt.Sprintf("%s - %s", record[1], record[2]),
			},
		)
	}

	return Unique(options)
}

func Unique[T comparable](s []T) []T {
	inResult := make(map[T]bool)
	var result []T
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

func Contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
