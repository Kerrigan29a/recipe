package main

import (
	"flag"
	"fmt"
	"github.com/Kerrigan29a/recipe"
	"os"
	"runtime"
)

var version string

func parseArgs(task *string, numWorkers *uint, level *recipe.LoggerLevel) []string {
	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println("")
		fmt.Printf("Version: %s\n", version)
	}
	var verbose, quiet bool
	flag.UintVar(numWorkers, "w", uint(runtime.NumCPU()), "Amount of workers")
	flag.StringVar(task, "m", "", "Main task")
	flag.BoolVar(&verbose, "v", false, "Show more information")
	flag.BoolVar(&quiet, "q", false, "Show less information")
	flag.Parse()
	paths := flag.Args()
	if len(paths) <= 0 {
		fmt.Fprintf(os.Stderr, "Must supply a recipe file\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if verbose && quiet {
		fmt.Fprintf(os.Stderr, "Only can select verbose or quiet mode, but not both\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if verbose {
		*level = recipe.DebugL
	} else if quiet {
		*level = recipe.WarningL
	} else {
		*level = recipe.InfoL
	}
	return paths
}

func main() {
	var task string
	var numWorkers uint
	var level recipe.LoggerLevel
	paths := parseArgs(&task, &numWorkers, &level)
	logger := recipe.NewLogger("[ Main ] ")
	logger.Level = level
	recipeLogger := recipe.NewLogger("[Recipe] ")
	recipeLogger.Level = level
	logger.Info("Version: %s", version)
	for _, path := range paths {
		recipe, err := recipe.Open(path, recipeLogger)
		if err != nil {
			logger.Fatal(err)
		}
		if task == "" {
			err = recipe.RunMain(numWorkers)
		} else {
			err = recipe.RunTask(task, numWorkers)
		}
		if err != nil {
			logger.Fatal(err)
		}
	}
}
