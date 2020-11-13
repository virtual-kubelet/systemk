package main

import (
	"context"
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
	fmt.printf("%+v\n", pods)
}
