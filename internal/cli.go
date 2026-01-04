package main

import (
	"fmt"
	"os"
)

// printErrorAndExit prints error e and exists the application
func printErrorAndExit(e error) {
	fmt.Println(e)
	os.Exit(1)
}

func main() {
	confPath, err := getConfigPath()
	if err != nil {
		printErrorAndExit(err)
	}

	conf, err := getConfig(confPath)
	if err != nil {
		printErrorAndExit(err)
	}

	runGui(conf)
}
