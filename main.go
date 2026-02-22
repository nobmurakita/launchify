package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	config, plist, err := RunApp()
	if err != nil {
		log.Fatal(err)
	}
	if config == nil {
		fmt.Println("キャンセルしました")
		return
	}
	if err := Install(config.Label, plist, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
