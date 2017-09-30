package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

var (
	domain   = flag.String("domain", "", "the hosting domain to provide authority/answers for")
	port     = flag.String("port", "8053", "the port to bind the service to, defaults to 8053")
	tcpOn    = flag.Bool("tcp", true, "provide service on port 8053/tcp, defaults to true")
	udpOn    = flag.Bool("udp", true, "provide service on port 8053/udp, defaults to true")
	compress = flag.Bool("compress", false, "compress replies")
)

func isCommand(checkCommand string) bool {
	// round robin
	if checkCommand == "RR" {
		return true
	}

	// fail percent
	if checkCommand[0:1] == "F" {
		i, err := strconv.ParseInt(checkCommand[1:len(checkCommand)], 10, 32)
		if err != nil || i > 99 {
			return false
		}
		return true
	}

	return false
}

// for a given address parses blocks of ips
// 127.0.0.1.domain.tld - one address ([127.0.0.1])
// 10.0.0.1.10.0.0.2.domain.tld - two addresses ([10.0.0.1,10.0.0.2])
// 10.0.0.1,2134:0000:1234:4567:2468:1236:2444:2106 - two addresses ([10.0.0.1,2134:0000:1234:4567:2468:1236:2444:2106])
// this uses the "."s as delimeters and jumps to the next one when parsing
// when it finds a match it discards that part of the string and keeps loking
// if the left index reaches the start of the string with no result and the right index is further away than "::"
// the right index will jump to the next dot. this prevents screwed up addresses from messing it up and also allows
// to do things that are easier to read like "10.0.0.1.and.10.5.4.1"
// one way to hose it up is still something like:
// 10.27.14.34.45.337.0.1 which will end up with one address: [27.14.34.45] which isn't the intent since 337 is probably a mistake
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
		checkIP := net.ParseIP(checkString)
		if checkIP != nil {
			responses = append(responses, checkIP)
			rightIndex = leftIndex - 1
			leftIndex = rightIndex - 1
		} else {
			// if we are already at 0, stop
			if leftIndex <= 0 {
				// try and skip something broken on the right
				if rightIndex > 2 {
					rightIndex = strings.LastIndex(addressString[0:rightIndex-1], ".")
					leftIndex = rightIndex - 1
				} else {
					break
				}
			} else {
				// use "." as the next jump point
				leftIndex = strings.LastIndex(addressString[0:leftIndex-1], ".")
				// jump/correct to start of string if no index found
				if leftIndex < 0 {
					leftIndex = 0
				} else {
					leftIndex++
				}
			}
		}
	}

	return responses
}

func respondToQuestion(w dns.ResponseWriter, request *dns.Msg, message *dns.Msg, q dns.Question) {
	var (
		rr   dns.RR
		ipV6 net.IP
		ipV4 net.IP
	)

	// parse current question
	fmt.Printf("Question: %s", q.Name)
	if q.Qtype == dns.TypeA {
		fmt.Print(" (A)\n")
	} else {
		fmt.Print(" (AAAA)\n")
	}
	questionName := q.Name

	// parse off the end domain and traling dot
	remainder := questionName[0 : len(questionName)-len(*domain)-1]

	// check for command
	command := ""
	lastDotIndex := strings.LastIndex(remainder, ".")
	if lastDotIndex > -1 {
		potentialCommand := strings.ToUpper(remainder[lastDotIndex+1 : len(remainder)])
		if isCommand(potentialCommand) {
			remainder = remainder[0:lastDotIndex]
			//fmt.Printf("Found command: %s, with remainder: %s\n", potentialCommand, remainder)
			command = potentialCommand
		}
	}

	// get list of IPs
	ips := parseIPs(remainder)

	// if no ips are available then no domain is found
	if len(ips) < 1 {
		return
	}

	// commands only need to execute if a command is found
	if len(command) > 0 {
		fmt.Printf("Setting command: %s\n", command)

		// fake "round" robin which is just a random distribution
		if command == "RR" && len(ips) > 1 {
			chosenRecordIndex := rand.Intn(len(ips))
			ips = ips[chosenRecordIndex : chosenRecordIndex+1]
		}

		// simulate failure command
		if command[0:1] == "F" {
			i, _ := strconv.ParseInt(command[1:len(command)], 10, 32)
			roll := rand.Intn(100)
			// if the roll exceeds the threshold, then do not respond
			if roll <= int(i) {
				ips = []net.IP{}
			}
		}
	}

	// for each IP respond
	for _, ip := range ips {
		// set values based on presence of ipv4/ipv6
		if ip != nil {
			ipV4 = ip.To4()
			ipV6 = ip.To16()
		}

		// create a record for the given response
		if q.Qtype == dns.TypeA && ipV4 != nil {
			rr = &dns.A{
				Hdr: dns.RR_Header{Name: questionName, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
				A:   ipV4,
			}
			message.Answer = append(message.Answer, rr)
		}

		if ipV6 != nil {
			rr = &dns.AAAA{
				Hdr:  dns.RR_Header{Name: questionName, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
				AAAA: ipV6,
			}
			if q.Qtype == dns.TypeAAAA {
				message.Answer = append(message.Answer, rr)
			}
		}
	}
}

func serve(netType string) {
	addr := "0.0.0.0:" + *port
	fmt.Printf("Starting %s server on address: %s ...\n", netType, addr)
	server := &dns.Server{Addr: addr, Net: netType, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the "+netType+" server: %s\n", err.Error())
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

	// set return code to NXDOMAIN if no answeres are found
	if len(m.Answer) < 1 {
		m.Rcode = dns.RcodeNameError
	}

	// write back message
	w.WriteMsg(m)

	// we will hang up when done
	//w.Close()
}

func checkDomain(domain string) bool {
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
	if !checkDomain(*domain) {
		// if a bad domain then exit
		fmt.Printf("The given domain (%s) is not a valid domain.\n", *domain)
		os.Exit(1)
	}

	// seed random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// set the function being used to handle the dns questions
	dns.HandleFunc(*domain, handleQuestions)

	// based on options/config decide what protocols to provide
	if *tcpOn {
		go serve("tcp")
	}
	if *udpOn {
		go serve("udp")
	}

	// wait for os signal
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}
