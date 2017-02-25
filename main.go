package main

import (
	"log"
	"net"

	"github.com/scgolang/osc"
)

func main() {
	laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:5610")
	if err != nil {
		log.Fatal(err)
	}
	srv, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	ch := make(chan chan uint64)
	go loop(ch, 1000)
	if err := srv.Serve(osc.Dispatcher{
		"/scids/next": func(m osc.Message) error {
			req := make(chan uint64)
			ch <- req
			return srv.SendTo(m.Sender, osc.Message{
				Address: "/scids/next",
				Arguments: osc.Arguments{
					osc.Int(<-req),
				},
			})
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func loop(ch chan chan uint64, i uint64) {
	for req := range ch {
		req <- i
		i++
		log.Printf("now at %d\n", i)
	}
}
