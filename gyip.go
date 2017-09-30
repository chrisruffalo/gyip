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

// for a given address parses blocks of ips
// 127.0.0.1.domain.tld - one address ([127.0.0.1])
// 10.0.0.1.10.0.0.2.domain.tld - two addresses ([10.0.0.1,10.0.0.2])
// 10.0.0.1,2134:0000:1234:4567:2468:1236:2444:2106 - two addresses ([10.0.0.1,2134:0000:1234:4567:2468:1236:2444:2106])
// this thing is very very opportunistic it will parse the _first_ time it matches and it _will not_ see how "far" the match
// goes.
func parseIPs(addressString string) []net.IP {
  // responses
  var responses []net.IP

  // start with the left and right comparison positions
  leftIndex := len(addressString) - 1
  rightIndex := len(addressString)

  // operate while the left index is still at least 0, meaning there is some string to work with
  for leftIndex >= 0 {
    checkString := addressString[leftIndex:rightIndex]
    //fmt.Printf("check string: %s\n", checkString)
    checkIp := net.ParseIP(checkString)
    if checkIp != nil {
      responses = append(responses, checkIp)
      rightIndex = leftIndex - 1
      leftIndex = rightIndex - 1
    } else {
      // if we are already at 0, stop
      if leftIndex == 0 {
        break
      }
      // use "." as the next jump point
      leftIndex = strings.LastIndex(addressString[0:leftIndex - 1],".")
      // jump/correct to start of string if no index found
      if leftIndex < 0 {
        leftIndex = 0
      } else {
        leftIndex++
      }

    }
  }

  return responses
}

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

  // parse off the end domain and traling dot
  remainder := questionName[0:len(questionName) - len(*domain) - 1]

  // get list of IPs
  ips := parseIPs(remainder)

  // if no ips are available then no domain is found
  if len(ips) < 1 {
    message.Rcode = dns.RcodeNameError
    return
  }

  // for each IP respond
  for _, ip := range ips {
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
