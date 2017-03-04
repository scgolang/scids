// Package scid exists to help ephemeral SuperCollider clients reliably generate
// unique synth ID's (which have to be int32).
package scid

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
)

const (
	AddrNext  = "/scids/next"
	AddrReply = "/reply"
	Port      = "5610"
)

var (
	ch   chan int32
	conn osc.Conn
)

func Next() (int32, error) {
	if err := conn.Send(osc.Message{Address: AddrNext}); err != nil {
		return 0, err
	}
	id := <-ch
	return id, nil
}

func init() {
	ch = make(chan int32)

	scidsConn, err := connect()
	if err != nil {
		log.Fatal(err)
	}
	conn = scidsConn

	go func(ch chan int32) {
		_ = scidsConn.Serve(osc.Dispatcher{
			AddrReply: osc.Method(func(m osc.Message) error {
				if expected, got := 2, len(m.Arguments); expected != got {
					return errors.Errorf("expected %d arguments, got %d", expected, got)
				}
				replyAddr, err := m.Arguments[0].ReadString()
				if err != nil {
					return err
				}
				if AddrNext != replyAddr {
					return nil
				}
				id, err := m.Arguments[1].ReadInt32()
				if err != nil {
					return errors.Wrap(err, "getting next synth ID")
				}
				ch <- id
				return nil
			}),
		})
	}(ch)
}

func connect() (osc.Conn, error) {
	listen, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", Port))
	if err != nil {
		return nil, err
	}
	scidsConn, err := osc.DialUDP("udp", listen, remote)
	if err != nil {
		return nil, err
	}
	return scidsConn, nil
}
