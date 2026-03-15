package main

import (
	"fmt"
	"github.com/RealFerdinandDeSaussure/flunder/internal"
	"os"
)

// printErrorAndExit prints error e and exists the application
func printErrorAndExit(e error) {
	fmt.Println(e)
	os.Exit(1)
}

func main() {
	confPath, err := flunder.GetConfigPath()
	if err != nil {
		printErrorAndExit(err)
	}

	conf, err := flunder.GetConfig(confPath)
	if err != nil {
		printErrorAndExit(err)
	}

	flunder.RunGui(conf)
}
