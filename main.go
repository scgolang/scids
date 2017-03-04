package main

import (
	"bytes"
	"flag"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"github.com/scgolang/sc"
	"github.com/scgolang/scids/scid"
)

func main() {
	var scsynthAddr string
	flag.StringVar(&scsynthAddr, "scsynth", "127.0.0.1:57120", "scsynth listening address")
	flag.Parse()

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

	scClient, err := newClient(scsynthAddr, ch)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Serve(osc.Dispatcher{
		scid.AddrNext: osc.Method(func(m osc.Message) error {
			req := make(chan int32)
			ch <- req
			return srv.SendTo(m.Sender, osc.Message{
				Address: scid.AddrReply,
				Arguments: osc.Arguments{
					osc.String(scid.AddrNext),
					osc.Int(<-req),
				},
			})
		}),
		scid.AddrSynthdef: scClient,
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

// client is a SuperCollider client.
type client struct {
	*sc.Client

	group *sc.Group
	ids   chan chan int32
}

// Handle handles a synthdef message.
func (c *client) Handle(m osc.Message) error {
	if expected, got := scid.AddrSynthdef, m.Address; expected != got {
		return errors.Errorf("expected %s, got %s", expected, got)
	}
	if expected, got := 1, len(m.Arguments); expected != got {
		return errors.Errorf("expected %d, got %d", expected, got)
	}
	buf, err := m.Arguments[0].ReadBlob()
	if err != nil {
		return errors.Wrap(err, "reading blob")
	}
	req := make(chan int32)

	c.ids <- req

	def, err := sc.ReadSynthdef(bytes.NewReader(buf))
	if err != nil {
		return errors.Wrap(err, "reading synthdef from buffer")
	}
	if err := c.SendDef(def); err != nil {
		return errors.Wrap(err, "sending synthdef")
	}
	_, err = c.group.Synth(def.Name, <-req, sc.AddToTail, nil)
	return errors.Wrap(err, "creating synth")
}

// newClient creates a new SuperCollider client.
func newClient(scsynthAddr string, ids chan chan int32) (*client, error) {
	c, err := sc.NewClient("udp", "0.0.0.0:0", scsynthAddr, 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "creating sc client")
	}
	g, err := c.AddDefaultGroup()
	if err != nil {
		return nil, errors.Wrap(err, "adding default group")
	}
	return &client{Client: c, group: g, ids: ids}, nil
}
