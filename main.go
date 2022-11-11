package main

import (
	"os"
	"log"
	"flag"
	"time"
	"io/fs"
	"strings"
	"runtime"
	"context"
	"os/exec"
	"golang.org/x/time/rate"
)

var (
	directory  string
	glob string
	ignore bool
	procs int
	command string
	options string
	poll time.Duration
)

func init() {
	flag.StringVar(&directory, "dir", ".", "directory to watch")
	flag.StringVar(&glob, "glob", "*", "globbing pattern for files to watch")
	flag.BoolVar(&ignore, "ignore", true, "ignore files already present in watched directories")
	flag.IntVar(&procs, "procs", runtime.NumCPU(), "max number of parallel processes")
	flag.StringVar(&command, "cmd", "", "command to trigger on each finished file")
	flag.StringVar(&options, "options", "", "comma-separated list of command options (use /_ to represent finished file)")
	flag.DurationVar(&poll, "poll", time.Second*5, "file polling period")
}

func main() {
	// parse commandline flags
	flag.Parse()

	// check that user-supplied command exists
	_, err := exec.LookPath(command)
	if err != nil {
		panic(err)
	}

	// check that the user-supplied directory exists
	checkDir, err := os.Stat(directory)
	if err != nil {
		panic(err)
	}
	if !checkDir.IsDir() {
		panic("is not a directory")
	}
	dir := os.DirFS(directory)	
	
	
	// limit max number of running goroutines
	runtime.GOMAXPROCS(procs)

	// setup logger
	log.SetFlags(log.Ltime)
	
	var queue = make(map[string]interface{})

	// populate queue with files that match at beginning of execution 
	if ignore {
		files, err := fs.Glob(dir, glob)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			queue[file] = struct{}{}
		}
	}

	limiter := rate.NewLimiter(rate.Every(poll), 1)
	
	// iterate forever on a loop
	for {
		// only check files in polling period
		if err := limiter.Wait(context.Background()); err != nil {
			panic(err)
		}

		// find matching files
		files, err := fs.Glob(dir, glob)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			// if file is in queue, it is already either
			// 1. ignored
			// 2. currently being monitored by a goroutine
			// 3. has already finished and been passed to the call program
			if _, exists := queue[file]; !exists {
				queue[file] = struct{}{}

				go func(f string) {
					// get initial file stats
					info, err := fs.Stat(dir, f)
					if err != nil {
						log.Print(err.Error())
						return
					}
					for {
						time.Sleep(poll)
						latestInfo, err := fs.Stat(dir, f)
						if err != nil {
							log.Printf("%s %s", info.Name(), err.Error())
							return
						}
						if latestInfo.Size() == info.Size() && latestInfo.ModTime() == info.ModTime() {
							cmd := exec.CommandContext(context.Background(), command, strings.Split(strings.ReplaceAll(options, "/_", info.Name()), ",")...)
							cmd.Dir = directory
							out, err := cmd.CombinedOutput()
							if err != nil {
								log.Printf("%s %s", info.Name(), err.Error())
								return
							}

							log.Printf("%s %s", info.Name(), string(out))
							break
						}
						info = latestInfo
					}
				}(file)
			}
		}
	}
}
