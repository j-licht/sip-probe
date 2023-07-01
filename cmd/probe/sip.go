package main

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/jart/gosip/sip"
	"github.com/jart/gosip/util"
)

func checkSIPOnline(name string) error {
	sock, err := net.Dial("udp", fmt.Sprintf("%s:5060", name))
	if err != nil {
		return err
	}
	defer sock.Close()
	raddr := sock.RemoteAddr().(*net.UDPAddr)
	laddr := sock.LocalAddr().(*net.UDPAddr)

	options := sip.Msg{
		CSeq:       util.GenerateCSeq(),
		CallID:     util.GenerateCallID(),
		Method:     "OPTIONS",
		CSeqMethod: "OPTIONS",
		Accept:     "application/sdp",
		UserAgent:  "pokémon/1.o",
		Request: &sip.URI{
			Scheme: "sip",
			User:   "echo",
			Host:   raddr.IP.String(),
			Port:   uint16(raddr.Port),
		},
		Via: &sip.Via{
			Version:  "2.0",
			Protocol: "UDP",
			Host:     laddr.IP.String(),
			Port:     uint16(laddr.Port),
			Param:    &sip.Param{Name: "branch", Value: util.GenerateBranch()},
		},
		Contact: &sip.Addr{
			Uri: &sip.URI{
				Host: laddr.IP.String(),
				Port: uint16(laddr.Port),
			},
		},
		From: &sip.Addr{
			Uri: &sip.URI{
				User: "gosip",
				Host: "justinetunney.com",
				Port: 5060,
			},
			Param: &sip.Param{Name: "tag", Value: util.GenerateTag()},
		},
		To: &sip.Addr{
			Uri: &sip.URI{
				Host: raddr.IP.String(),
				Port: uint16(raddr.Port),
			},
		},
	}

	var b bytes.Buffer
	options.Append(&b)
	if amt, err := sock.Write(b.Bytes()); err != nil || amt != b.Len() {
		return err
	}

	memory := make([]byte, 2048)
	sock.SetDeadline(time.Now().Add(time.Second))
	amt, err := sock.Read(memory)
	if err != nil {
		return err
	}

	msg, err := sip.ParseMsg(memory[0:amt])
	if err != nil {
		return err
	}

	if !msg.IsResponse() || msg.Status != 200 || msg.Phrase != "OK" {
		return fmt.Errorf("not OK :[")
	}
	if options.CallID != msg.CallID {
		return fmt.Errorf("CallID didnt match")
	}
	if options.CSeq != msg.CSeq || options.CSeqMethod != msg.CSeqMethod {
		return fmt.Errorf("CSeq didnt match")
	}
	if options.From.String() != msg.From.String() {
		return fmt.Errorf("from headers didn't match:\n%s\n%s\n\n%s", options.From, msg.From, memory[0:amt])
	}
	if msg.To.Param.Get("tag") == nil {
		return fmt.Errorf("remote UA didnt tag To header:\n%s\n\n%s", msg.To, memory[0:amt])
	}
	msg.To.Param = nil
	if options.To.String() != msg.To.String() {
		return fmt.Errorf("to headers didn't match:\n%s\n%s\n\n%s", options.To, msg.To, memory[0:amt])
	}

	return nil
}