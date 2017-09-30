package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
  "strings"

	"github.com/miekg/dns"
)

var (
  domain     = flag.String("domain", "", "the hosting domain to provide authority/answers for")
  port       = flag.String("port", "8053", "the port to bind the service to, defaults to 8053")
  tcp_on     = flag.Bool("tcp", true, "provide service on port 8053/tcp, defaults to true")
  udp_on     = flag.Bool("udp", true, "provide service on port 8053/udp, defaults to true")
	compress   = flag.Bool("compress", false, "compress replies")
)

func respondToQuestion(w dns.ResponseWriter, request *dns.Msg, message *dns.Msg, q dns.Question) {
  var (
    rr dns.RR
  )

  // parse current question
  fmt.Printf("Question: %s", q.Name)
  if(q.Qtype == dns.TypeA) {
    fmt.Print(" (A)\n")
  } else {
    fmt.Print(" (AAAA)\n")
  }
  questionName := q.Name

  // parse off the end domain and DO NOT REMOVE trailing dot
  remainder := questionName[0:len(questionName) - len(*domain)]

  // now we need to determine if the remaineder is in ipv4 or ipv6 format
  // the easiest way to do that is to try and parse the IP over and over
  // again and see if anything happens. We can speed this up by continually
  // jumping to the next '.'
  var ip net.IP = nil
  for strings.Index(remainder, ".") >= 0 {
    //fmt.Printf("Remainder: %s\n", remainder[0:len(remainder) - 1])
    checkIp := net.ParseIP(remainder[0:len(remainder) - 1])

    // if the ip is a valid ip, use and stop
    if checkIp != nil {
      ip = checkIp
      break
    }

    // otherwise jump to next .
    remainder = remainder[strings.Index(remainder, ".")+1:len(remainder)]
  }

  if ip == nil {
    message.Rcode = dns.RcodeNameError
    return
  }

  // set values based on presence of ipv4/ipv6
  var ip_v4 net.IP = nil
  var ip_v6 net.IP = nil
  if ip != nil {
    ip_v4 = ip.To4()
    ip_v6 = ip.To16()
  }

  // create a record for the given response
  if q.Qtype == dns.TypeA && ip_v4 != nil {
    rr = &dns.A{
      Hdr: dns.RR_Header{Name: questionName, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
      A: ip_v4,
    }
    message.Answer = append(message.Answer, rr)
  }

  if ip_v6 != nil {
    rr = &dns.AAAA{
      Hdr: dns.RR_Header{Name: questionName, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
      AAAA: ip_v6,
    }
    if q.Qtype == dns.TypeAAAA {
      message.Answer = append(message.Answer, rr)
    } else {
      message.Extra = append(message.Extra, rr)
    }
  }
}

func serve(net_type string) {
  addr := "0.0.0.0:" + *port
  fmt.Printf("Starting %s server on address: %s ...\n", net_type, addr)
  server := &dns.Server{Addr: addr, Net: net_type, TsigSecret: nil}
  if err := server.ListenAndServe(); err != nil {
    fmt.Printf("Failed to setup the " + net_type + " server: %s\n", err.Error())
  }
}

func handleQuestions(w dns.ResponseWriter, r *dns.Msg) {
  // setup outbound message
  m := new(dns.Msg)
  m.SetReply(r)
  m.Compress = *compress
  m.Authoritative = true

  // todo: something if len(m.Question < 1)

  // handle _each_ question
  for _, q := range m.Question {
    // only "answer" if question is A or AAAA
    if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
      respondToQuestion(w, r, m, q)
    }
  }

  // write back message
  w.WriteMsg(m)

  // we will hang up when done
  //w.Close()
}

func check_domain(domain string) bool {
  // the domain must not be empty
  if domain == "" {
    return false
  }

  // todo: check if is actually domain shaped

  // passes all checks
  return true
}

func main() {
  // parse options
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

  // check domain
  if !check_domain(*domain) {
    // if a bad domain then exit
    fmt.Printf("The given domain (%s) is not a valid domain.\n", *domain)
    os.Exit(1)
  }

  // set the function being used to handle the dns questions
	dns.HandleFunc(*domain, handleQuestions)

  // based on options/config decide what protocols to provide
  if *tcp_on {
	   go serve("tcp")
  }
  if *udp_on {
	   go serve("udp")
  }

  // wait for os signal
  sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}
