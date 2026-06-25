package main

import (
	"time"
)

func main() {
	go StartServer()

	time.Sleep(2 * time.Second)

	Run()
}
