package main

import (
	"context"
	"log"

	"github.com/miekg/vks/pkg/provider"
)

func main() {
	p, err := provider.New()
	if err != nil {
		log.Fatal(err)
	}
	if err := p.GetPods(context.Background()); err != nil {
		log.Fatal(err)
	}
}
