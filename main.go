package main

import (
	"context"
	"fmt"
	"log"

	"github.com/miekg/vks/systemd"
)

func main() {
	p, err := systemd.NewProvider()
	if err != nil {
		log.Fatal(err)
	}
	pods, err := p.GetPods(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", pods)
}
