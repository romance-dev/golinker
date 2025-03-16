package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	fmt.Println(runtime.Version(), "sample cmd line args: ", os.Args[1:])
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		switch sig {
		case os.Interrupt:
			fmt.Println("sample: SIGINT")
			os.Exit(130)
		case syscall.SIGTERM:
			fmt.Println("sample: SIGTERM")
			os.Exit(143)
		}
	}()

	rand.Seed(time.Now().UnixNano())

	i := 0
	for {
		i++
		fmt.Println("Message:", i)
		n := rand.Intn(2000)
		time.Sleep(time.Duration(n) * time.Millisecond)
	}
}
