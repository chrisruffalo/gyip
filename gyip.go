package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/chrisruffalo/gyip/command"
	"github.com/miekg/dns"
)

// regexp expression to match hostnames
var validHostnameRegexMatcher, _ = regexp.Compile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$")
var servingDomains = []string{}

var (
	hosts    = flag.String("host", "0.0.0.0", "The host to bind to. Can be a comma-seperated list of hosts. (Ex: \"--host 127.0.0.1,10.0.0.1\")")
	domain   = flag.String("domain", "", "Required. The hosting domain to provide authority/answers for. Can be a comma-separated list of domains. (Ex: \"--domain gyip.io,gyip.net\")")
	port     = flag.String("port", "8053", "The port to bind the service to (tcp and udp), defaults to 8053")
	tcpOff   = flag.Bool("tcpOff", false, "Disable listening on TCP, defaults to false")
	udpOff   = flag.Bool("udpOff", false, "Disable listening on UDP, defaults to false")
	compress = flag.Bool("compress", false, "Compress replies, defaults to false")
)

func reverse(ips []net.IP) {
	for i, j := 0, len(ips)-1; i < j; i, j = i+1, j-1 {
		ips[i], ips[j] = ips[j], ips[i]
	}
}

// for a given address parses blocks of ips
// 127.0.0.1.domain.tld - one address ([127.0.0.1])
// 10.0.0.1.10.0.0.2.domain.tld - two addresses ([10.0.0.1,10.0.0.2])
// 10.0.0.1.2134:0000:1234:4567:2468:1236:2444:2106.domain.tld - two addresses ([10.0.0.1,2134:0000:1234:4567:2468:1236:2444:2106])
// this uses the "."s as delimeters and jumps to the next one when parsing
// when it finds a match it discards that part of the string and keeps loking
// if the left index reaches the start of the string with no result and the right index is further away than "::"
// the right index will jump to the next dot. this prevents screwed up addresses from messing it up and also allows
// to do things that are easier to read like "10.0.0.1.and.10.5.4.1"
// one way to confuse the parser is to do something like:
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

	// since we worked right to left we need to reverse the order before responding
	// so that it maintains the left to right order we expect
	if len(responses) > 0 {
		reverse(responses)
	}

	return responses
}

// adapts the dns question to a response. this method is the bare minimum and allows a unit-testable
// point within the dns "resolution" pipe
func frameResponse(ip net.IP, questionType uint16, questionName string, currentQuestionDomain string) []dns.RR {
	var (
		records []dns.RR
		ipV6    net.IP
		ipV4    net.IP
	)

	// guards test cases
	if "" == questionName || strings.LastIndex(questionName, currentQuestionDomain) < 0 {
		return nil
	}

	// parse off the end domain and trailing dot
	remainder := questionName[0 : len(questionName)-len(currentQuestionDomain)-1]

	// ttl for response
	var ttl uint32
	// ips is am empty array
	var ips []net.IP

	// check for echo/reflect request
	if "echo" == strings.ToLower(remainder) || "reflect" == strings.ToLower(remainder) {
		ips = []net.IP{ip}
	} else {
		// check for command
		var cmd command.Command = command.Noop{}
		lastDotIndex := strings.LastIndex(remainder, ".")
		if lastDotIndex > -1 {
			potentialCommand := strings.ToUpper(remainder[lastDotIndex+1 : len(remainder)])
			cmd = command.New(potentialCommand)
			if cmd.Type() != command.NOOP {
				remainder = remainder[0:lastDotIndex]
			}
		}

		// get list of IPs
		ips = parseIPs(remainder)

		// if no ips are available then no domain is found
		if len(ips) < 1 {
			return nil
		}

		// use transform from found command and set the
		// ttl based on the transformation
		ips, ttl = cmd.Execute(ips)
	}

	// for each IP create a response record
	for _, ip := range ips {
		// set values based on presence of ipv4/ipv6
		if ip != nil {
			ipV4 = ip.To4()
			ipV6 = ip.To16()
		}

		// allocate new dns.RR for each loop
		var rr dns.RR

		// create a record for the given response
		if questionType == dns.TypeA && ipV4 != nil {
			rr = &dns.A{
				Hdr: dns.RR_Header{Name: questionName, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
				A:   ipV4,
			}
			records = append(records, rr)
		}

		if questionType == dns.TypeAAAA && ipV6 != nil {
			rr = &dns.AAAA{
				Hdr:  dns.RR_Header{Name: questionName, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
				AAAA: ipV6,
			}
			records = append(records, rr)
		}
	}

	return records
}

// takes dns-level information and does some work to adapt it to a framed question that can be "resolved"
func respondToQuestion(w dns.ResponseWriter, request *dns.Msg, message *dns.Msg, q dns.Question) {
	questionName := q.Name

	currentQuestionDomain := ""
	// find current domain
	for _, servedDomain := range servingDomains {
		if strings.HasSuffix(questionName, servedDomain) {
			currentQuestionDomain = servedDomain
			break
		}
	}

	// get ip
	var ip net.IP
	if cip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		ip = cip.IP
	}
	if cip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		ip = cip.IP
	}

	// parse log of current question
	qtype := ""
	if q.Qtype == dns.TypeA {
		qtype = "A"
	} else {
		qtype = "AAAA"
	}
	fmt.Printf("[%s] Question (%s): %s (%s)\n", ip, currentQuestionDomain, q.Name, qtype)

	response := frameResponse(ip, q.Qtype, questionName, currentQuestionDomain)
	if response != nil && len(response) > 0 {
		for _, rr := range response {
			message.Answer = append(message.Answer, rr)
		}
	}
}

// provides the envelope to handle the dns response from the DNS server api
func handleQuestions(w dns.ResponseWriter, r *dns.Msg) {
	// setup outbound message
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = *compress
	m.Authoritative = true

	// handle _each_ question
	for _, q := range m.Question {
		// only "answer" if question is A or AAAA
		if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
			respondToQuestion(w, r, m, q)
		}
	}

	// set return code to NXDOMAIN if no answers are found
	if len(m.Answer) < 1 {
		m.Rcode = dns.RcodeNameError
	}

	// write back message
	w.WriteMsg(m)
}

func serve(netType string, host string) {
	addr := host + ":" + *port
	fmt.Printf("Starting %s server on address: %s ...\n", netType, addr)
	server := &dns.Server{Addr: addr, Net: netType, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the %s server: %s\n", netType, err.Error())
	}
}

func main() {
	// parse options
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	if *domain == "" {
		fmt.Print("At least one domain to host is required.\n")
		os.Exit(1)
	}

	// split the input domain list
	hostingDomains := splitHosts(*hosts)
	// and do the same for the domains
	servingDomains = splitDomains(*domain)

	// check domain
	if len(servingDomains) < 1 {
		// if a bad domain then exit
		fmt.Print("No valid domains were given. The server will not start.\n")
		os.Exit(1)
	}

	// can't do anything if both tcp and udp are off
	if *tcpOff && *udpOff {
		fmt.Print("The options tcpOff and udpOff cannot both be set at the same time.\n")
		os.Exit(1)
	}

	// seed random number generator (does not need crypto-strength)
	// just used for `rr` and `f` commands
	rand.Seed(time.Now().UTC().UnixNano())

	// set the function being used to handle the dns questions
	for _, servingDomain := range servingDomains {
		// log start of service
		fmt.Printf("Providing service for domain: %s\n", servingDomain)
		dns.HandleFunc(servingDomain, handleQuestions)
	}
	fmt.Print("(All other domains will receive NOZONE response)\n")

	// for all other domains just return a not zone response
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Compress = *compress

		// just say that the response code is that the question isn't in the zone
		m.Rcode = dns.RcodeNotZone

		// write back message
		w.WriteMsg(m)
	})

	// based on options/config decide what protocols to provide
	for _, host := range hostingDomains {
		if strings.Index(host, ":") >= 0 {
			host = "[" + host + "]"
		}

		if !*tcpOff {
			go serve("tcp", host)
		}
		if !*udpOff {
			go serve("udp", host)
		}
	}

	// wait for os signal
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}

func splitHosts(hostInput string) []string {
	outputHosts := []string{}

	hostStringSplit := strings.Split(hostInput, ",")
	for _, hostToCheck := range hostStringSplit {
		// if empty, continue
		if hostToCheck == "" {
			continue
		}

		// if ip, add to list
		actualIP := net.ParseIP(hostToCheck)
		if actualIP != nil {
			// if someone uses the literal 0.0.0.0 just use that and stop parsing
			if actualIP.String() == "0.0.0.0" {
				return []string{"0.0.0.0"}
			}

			outputHosts = append(outputHosts, actualIP.String())
			continue
		}

		// if domain attempt resolution and, add to list
		if checkDomain(hostToCheck) {
			resolvedIPs, err := net.LookupIP(hostToCheck)
			if err == nil {
				for _, rIP := range resolvedIPs {
					outputHosts = append(outputHosts, rIP.String())
				}
			}
		}
	}

	// if no hosts found just bind to everything (no specified host)
	// shouldn't happen but it could in theory if no valid hosts are
	// specified
	if len(outputHosts) < 1 {
		return []string{"0.0.0.0"}
	}

	return outputHosts
}

func splitDomains(domainInput string) []string {
	outputDomains := []string{}

	// split the input domain list
	domainStringSplit := strings.Split(domainInput, ",")
	for _, domainToCheck := range domainStringSplit {
		// trim whitespace
		domainToCheck = strings.TrimSpace(domainToCheck)
		// string needs contents or else we just go to next entry
		if domainToCheck == "" {
			continue
		}
		// set up proper DNS end . (to keep in array, removed for check by validation function)
		if string(domainToCheck[len(domainToCheck)-1]) != "." {
			domainToCheck = domainToCheck + "."
		}
		// if the domain is ok keep it otherwise put some errors so that the end-user knows
		if checkDomain(domainToCheck) {
			outputDomains = append(outputDomains, domainToCheck)
		} else {
			fmt.Printf("The domain \"%s\" is not a valid domain and cannot be served\n", domainToCheck)
		}
	}

	return outputDomains
}

func checkDomain(domain string) bool {
	// the domain must not be empty
	if domain == "" {
		return false
	}

	// we need to remove trailing '.' for the regexp to work
	if string(domain[len(domain)-1]) == "." {
		domain = domain[0 : len(domain)-1]
	}

	// not a valid domain name
	if !validHostnameRegexMatcher.MatchString(domain) {
		return false
	}

	// passes all checks
	return true
}
