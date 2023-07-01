package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/domainr/dnsr"
	"github.com/miekg/dns"
)

var re = regexp.MustCompile(`(?m)(\d+)\s+(\d+)\s+(\d+)\s+(.+)`)

var currentOnly string

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, currentOnly))
			if err != nil {
				log.Fatal(err)
			}
			m.Answer = append(m.Answer, rr)
		}
	}

	w.WriteMsg(m)
}

func main() {
	port := flag.Int("port", 5053, "listen port for the dns server")
	interval := flag.Duration("interval", time.Minute, "interval for checking SIP")
	flag.Parse()

	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	server := &dns.Server{Addr: ":" + strconv.Itoa(*port), Net: "udp"}
	log.Printf("Starting at %d\n", port)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	r := dnsr.NewResolver(dnsr.WithCache(100))

	ticker := time.NewTicker(*interval)
	for {
		entries := r.Resolve("_sip._udp.tel.t-online.de", "SRV")
		_, entries2, err := net.LookupSRV("sip", "udp", "tel.t-online.de")
		if err != nil {
			log.Fatal(err)
		}
		for _, e := range entries {
			log.Println(e.Value)
			matches := re.FindStringSubmatch(e.Value)
			if len(matches) != 5 {
				log.Printf("cannot parse SRV record %s\n", e.Value)
				continue
			}

			log.Println(matches[4])
			err := checkSIPOnline(matches[4])
			if err != nil {
				log.Println(err)
				continue
			}

			log.Println("looks OK")

			ip, err := net.ResolveIPAddr("ip4", matches[4])
			if err != nil {
				log.Println(err)
				continue
			}
			currentOnly = ip.String()
		}

		for _, e := range entries2 {
			log.Printf("weight: %d, priority %d %s",e.Weight, e.Priority, e.Target)
			err := checkSIPOnline(e.Target)
			if err != nil {
				log.Println(err)
				continue
			}

			log.Println("looks OK")

			ip, err := net.ResolveIPAddr("ip4", e.Target)
			if err != nil {
				log.Println(err)
				continue
			}
			currentOnly = ip.String()
		}
		<-ticker.C
	}

}
