package main

import (
	"net"
	"reflect"
	"testing"

	"github.com/miekg/dns"
)

func TestDomainCheck(t *testing.T) {

	data := []struct {
		domain   string
		expected bool
	}{
		{"", false},
		{"google.com", true},
		{"127.0.0.1.gyip.io", true},
		{"fort@gyip.io", false},
		{"really.long.domain.with.lots.of.dots.should.still.work", true},
	}

	for _, item := range data {
		if checkDomain(item.domain) != item.expected {
			t.Errorf("The string '%s' was not evaluated as expected (was: %t, expected: %t)", item.domain, checkDomain(item.domain), item.expected)
		}
	}
}

func TestIPResponses(t *testing.T) {

	data := []struct {
		dnsType        uint16
		questionDomain string
		inputQuestion  string
		outputIPs      []string
	}{
		// IPV4
		{dns.TypeA, "gyip.io", "127.0.0.1.gyip.io", []string{"127.0.0.1"}},
		{dns.TypeA, "wrong.io", "127.0.0.1.gyip.io", []string{}},
		{dns.TypeA, "wrong.io", "", []string{}},
		{dns.TypeA, "", "wrong.io", []string{}},
		{dns.TypeA, "", "", []string{}},
		{dns.TypeA, "gyip.io", "alpha.domain.127.0.0.1.gyip.io", []string{"127.0.0.1"}},
		{dns.TypeA, "gyip.io", "alpha.domain.127.0.0.1.10.0.0.24.gyip.io", []string{"127.0.0.1", "10.0.0.24"}},
		{dns.TypeA, "gyip.io", "domain.127.0.0.gyip.io", []string{}},
		{dns.TypeA, "gyip.io", "really.long27.498.&.confusing.483.2383.3838.455.12.127.0.0.1.312.gyip.io", []string{"127.0.0.1"}},
		{dns.TypeA, "gyip.io", "virthost10.10.1.1.1.gyip.io", []string{"10.1.1.1"}},
		{dns.TypeA, "gyip.io", "virthost.10.10.1.1.gyip.io", []string{"10.10.1.1"}},
		{dns.TypeA, "gyip.io", "()())()).10.0.0.1.).gyip.io", []string{"10.0.0.1"}},
		{dns.TypeA, "gyip.io", ".gyip.io", []string{}},
		{dns.TypeA, "gyip.io", "*(&()()*#@&#$)(*#_+__)(@_(@()@>........904098......)).gyip.io", []string{}},
		// with a command but don't inspect command implementation
		{dns.TypeA, "gyip.io", "10.0.0.1.rr.gyip.io", []string{"10.0.0.1"}},
		// IPV6
		{dns.TypeAAAA, "gyip.io", "::1.gyip.io", []string{"::1"}},
		{dns.TypeAAAA, "wrong.io", "::1.gyip.io", []string{}},
		{dns.TypeAAAA, "domain.tld", "10.0.0.1.2134:0000:1234:4567:2468:1236:2444:2106.domain.tld", []string{"10.0.0.1", "2134:0000:1234:4567:2468:1236:2444:2106"}},
		{dns.TypeAAAA, "domain.tld", "2134:0000:1234:4567:2468:1236:2444:2106.domain.tld", []string{"2134:0000:1234:4567:2468:1236:2444:2106"}},
		{dns.TypeAAAA, "domain.tld", "2134:0000:1234:4567:2468:1236:2444:2106.2134:0000:1234:4567:2468:1236:2444:2106.domain.tld", []string{"2134:0:1234:4567:2468:1236:2444:2106", "2134:0000:1234:4567:2468:1236:2444:2106"}},
	}

	for _, item := range data {
		records := frameResponse(nil, item.dnsType, item.inputQuestion, item.questionDomain)
		// face check to see if records have the expected length
		if len(item.outputIPs) != len(records) {
			t.Errorf("The query '%s' for domain '%s' did not return the expected number of records (returned %d, expected %d", item.inputQuestion, item.questionDomain, len(records), len(item.outputIPs))
			continue
		}
		actualFound := []string{}
		for _, record := range records {
			var dnsIP net.IP

			switch item.dnsType {
			case dns.TypeA:
				dnsIP = record.(*dns.A).A
			case dns.TypeAAAA:
				dnsIP = record.(*dns.AAAA).AAAA
			}
			dnsIPString := dnsIP.String()
			actualFound = append(actualFound, dnsIPString)
		}

		notFound := []string{}
		// now inspect records to ensure expected data is found
		for _, expectedIP := range item.outputIPs {
			// we do this to the expected ip because it normalizes ipv6 strings between expected and actual
			expectedIP = net.ParseIP(expectedIP).String()
			found := false
			for _, foundIP := range actualFound {
				if foundIP == expectedIP {
					found = true
					break
				}
			}
			if !found {
				notFound = append(notFound, expectedIP)
			}
		}

		if len(notFound) > 0 {
			t.Errorf("The query '%s' for domain '%s' did not find expected elements %v (was %v, expected %v)", item.inputQuestion, item.questionDomain, notFound, actualFound, item.outputIPs)
		}
	}
}

func TestDomainSplit(t *testing.T) {
	data := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"                                                         ", []string{}},
		{"gyip.io", []string{"gyip.io."}},
		{"gyip.io.", []string{"gyip.io."}},
		{"gyip.io,,,,,dog.com", []string{"gyip.io.", "dog.com."}},
		{" gyip.io , dog.com ", []string{"gyip.io.", "dog.com."}},
		{"gyip.io,)))___adsfasdf____,dog.com ", []string{"gyip.io.", "dog.com."}},
	}

	for _, item := range data {
		domainResult := splitDomains(item.input)
		if !reflect.DeepEqual(item.expected, domainResult) {
			t.Errorf("The domain input '%s' was not properly split (was: %v, expected %v)", item.input, domainResult, item.expected)
		}
	}
}

func TestHostSplit(t *testing.T) {
	data := []struct {
		input    string
		expected []string
	}{
		{"", []string{"0.0.0.0"}},
		{"10.0.0.1", []string{"10.0.0.1"}},
		{"127.0.0.1,12.4.3.9,888.888.888.888,10.1.1.1", []string{"127.0.0.1", "12.4.3.9", "10.1.1.1"}},
		{"127.0.0.1,12.4.3.9,888.888.888.888,10.1.1.1,0.0.0.0", []string{"0.0.0.0"}},
		{"localhost", []string{"::1", "127.0.0.1"}},
	}

	for _, item := range data {
		hostResult := splitHosts(item.input)
		if !reflect.DeepEqual(item.expected, hostResult) {
			t.Errorf("The host input '%s' was not properly split (was: %v, expected %v)", item.input, hostResult, item.expected)
		}
	}
}
