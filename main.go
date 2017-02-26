package main

import (
	"log"
	"net"

	"github.com/scgolang/osc"
	"github.com/scgolang/scids/scid"
)

func main() {
	laddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("0.0.0.0", scid.Port))
	if err != nil {
		log.Fatal(err)
	}
	srv, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	ch := make(chan chan int32)
	go loop(ch, 1000)
	if err := srv.Serve(osc.Dispatcher{
		scid.AddrNext: func(m osc.Message) error {
			req := make(chan int32)
			ch <- req
			return srv.SendTo(m.Sender, osc.Message{
				Address: scid.AddrReply,
				Arguments: osc.Arguments{
					osc.String(scid.AddrNext),
					osc.Int(<-req),
				},
			})
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func loop(ch chan chan int32, i int32) {
	for req := range ch {
		req <- i
		i++
	}
}
