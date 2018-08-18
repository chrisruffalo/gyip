# GYIP - G(et) Y(our) IP

## Overview
GYIP is a special DNS server that allows you to ask questions you already know the answer to. It provides a global way to resolve address you already know so that things like virtual hosts can work with name resolution. It is inspired by the **excellent** [nip.io](http://nip.io) and [xip.io](http://xip.io). The main difference is that those services are based on PowerDNS and GYIP is a simple Go program. GYIP is more suited to sites and locations without their own DNS infrastructure. With GYIP you can create a DNS server for a specific domain that can easily answer DNS questions that point to an IP, a list of IPs, or with even other special conditions.

In contrast to XIP and NIP: GYIP is _probably_ not production ready. What it provides in additional features is offset by the fact that it is new, untested, and not based on something that has been battle-hardened over years of testing in various environments.

## Usage

### Build
GYIP can be built like any other Go program. There are ways to set the version and make the build static. The `build.sh` and `docker.sh` scripts go over that in more detail.
```bash
[]$ go build ./gyip.io
```

### Execution
The resulting binary requires a domain option to function. This domain will be the root domain for any question you ask. The trailing '.' is required or the underlying domain will not be resolved. By default the server starts listenting on 8053/tcp and 8053/udp. This allows the server to be run as a non-root user.
```bash
[]$ ./gyip --domain my.custom.io
```

## Container
The container is published [here](https://hub.docker.com/r/chrisruffalo/gyip/) and can be used in a way similar to any of the commands given in this guide. The `gyip` command is the default entrypoint and so all additional arguments are passed to it.
```bash
[]$ docker run -ti -p 8053:8053/tcp -p 8053:8053/udp chrisruffalo/gyip --domain gyip.io --host 0.0.0.0 --port 8053
```
It is important to note that whatever port is forwarded must be forwarded for **both** TCP and UDP protocols for the container to respond to most clients.

### Queries
The questions you ask GYIP allow name resolution of IP addresses as subrecords in the domain. (All of the examples in this document assume the use of `gyip.io` as the hosting domain.)
```bash
[]$ dig -p 8053 127.0.0.1.gyip.io @localhost +short
127.0.0.1
```

You can also ask with various subdomains of the same IP.
```bash
[]$ dig -p 8053 subdomain1.127.0.0.1.gyip.io @localhost +short
127.0.0.1
[]$ dig -p 8053 subdomain2.127.0.0.1.gyip.io @localhost +short
127.0.0.1
[]$ dig -p 8053 subsub1.subdomain3.127.0.0.1.gyip.io @localhost +short
127.0.0.1
```

Each subdomain returns the same IP in the IP list. What this effectively means is that you can run a local HTTP server or something more complicated (like OpenShift) that relies on virtual hosts and still get differentiated name resolution at a local IP. Or even a VM on a private network.

```bash
[]$ dig -p 8053 virtualhost.192.168.122.43.gyip.io @localhost +short
127.0.0.1
```

If you ask a question outside of the domain you will get a NOZONE response. If you ask a non-IP based query or a query with no IP component you get a NXDOMAIN response.
```bash
[]$ nslookup -port=8053 google.com localhost
Server:		localhost
Address:	::1#8053

** server can\'t find google.com: NOTZONE

[]$ nslookup -port=8053 torque.gyip.io localhost
Server:		localhost
Address:	::1#8053

** server can\'t find torque.gyip.io: NXDOMAIN
```

## Options
The `gyip` command accepts several options:
* **domain** - the domain that the server should respond to, can accept a single domain or comma-separated list of domains. (required, no default)
* **port** - the port the DNS server should listen on. this applies to both TCP and UDP. (default: 8053)
* **tcpOff** - set this option to turn off listening on the TCP protocol (default: false)
* **udpOff** - set this option to turn off listening on the UDP protocol (default: false)
* **compress** - set this option to compress DNS query responses (default: false)

Change the port:
```bash
[]$ ./gyip --domain gyip.io --port 53
```

Turn TCP off:
```bash
[]$ ./gyip --domain gyip.io --tcpOff
```

Compress responses:
```bash
[]$ ./gyip --domain gyip.io --compress
```

Multiple domains:
```bash
[]$ ./gyip --domain gyip.io,gyip.com,gyip.org
```

## Advanced Usage
The GYIP DNS responder was built with the idea that there would be some advanced features and functionality. It supports multiple IP addresses, IPv6, and various special commands. These optionas are intended to provide flexibility in domain resolution for your application needs.

### Reflective / Echo Query
The gyip server can return what it thinks the IP of the client is (the nearest gateway) which basically functions like a sort of "what's my outside ip". Using this would be helpful if you wanted to connect to something on the edge of your network without necessarily having to hard code that edge. Perhaps this would be useful for multiple sites. Or maybe you just want a DNS-based "what's my ip" query. This functionality is accessed by using the keyword `echo.<domian>` or `reflect.<domain>`. This functionality does not work with other commands.
```bash
[]$ dig -p 8053 echo.gyip.io @localhost +short A -b 127.0.0.1
127.0.0.1
[]$ dig -p 8053 reflect.gyip.io @localhost +short AAAA
::1
[]$ dig -p 8053 echo.gyip.io @localhost +short AAAA -b 127.0.0.1
::ffff:127.0.0.1
```
Note, when using local testing it is sensitive to the source interface. If the source is IPv6 (::1) it would return no response unless you asked for an AAAA record. It does, however, automatically convert IPv4 records for a AAAA record request if that's all that's available. (This is the same behavior as other queries.)

### Multiple Addresses
The query can contain multiple IP addresses and will return multiple A records.
```bash
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.gyip.io @localhost +short A
10.0.0.1
10.0.0.2
10.0.0.3
```
Notice that they are always returned in order from left to right as they would be read.

### IPv6
The ability to ask for AAAA (IPV6) records is built into GYIP so that it can respond to requests for that information. This can be useful when locally testing IPV6 resources that don't have domain names. This can take the form of an actual IPV6 request _or_ converting an IPV4 address to IPV6. Also notice that both short formats as well as long format IPV6 work.

```bash
[]$ dig -p 8053 127.0.0.1.gyip.io @localhost +short AAAA
::ffff:127.0.0.1
[]$ dig -p 8053 ::1.gyip.io @localhost +short AAAA
::1
[]$ dig -p 8053 2001:0db8:85a3:0000:0000:8a2e:0370:7334.gyip.io @localhost +short AAAA
2001:db8:85a3::8a2e:370:7334
[]$ dig -p 8053 2001:db8:85a3:0:0:8a2e:370:7334.gyip.io @localhost +short AAAA
2001:db8:85a3::8a2e:370:7334
[]$ dig -p 8053 2001:db8:85a3::8a2e:370:7334.gyip.io @localhost +short AAAA
2001:db8:85a3::8a2e:370:7334
```

This also works with _multiple_ IPV6 addresses (even when adding/converting an IPV4 address) and multiple AAAA records can be returned at the same time (multiple AAAA records) the same way that IPV4 addresses can.
```bash
[]$ dig -p 8053 2001:0db8:85a3:cdef:0000:8a2e:0431:7334.2001:0db8:85a3:0000:0000:8a2e:0370:7334.gyip.io @localhost +short AAAA
2001:db8:85a3:cdef:0:8a2e:431:7334
2001:db8:85a3::8a2e:370:7334
[]$ dig -p 8053 10.0.0.1.2001:0db8:85a3:0000:0000:8a2e:0370:7334.gyip.io @localhost +short AAAA
::ffff:10.0.0.1
2001:db8:85a3::8a2e:370:7334
```

Finally, if you ask for an A record with only IPV6 addresses you get an empty response.

### Commands
Individual commands can be issued by utilizing the first subdomain after the serving/host domain. This would be of the form `<question>.<command>.<domain>`. If you were serving "gyip.io" and your question was "sub.127.0.0.1" a query with a command in it would look like `sub.127.0.0.1.<command>.gyip.io`.

#### Random Robin
Sometimes the ability to reliably test multiple endpoints is required. Using the `rr` command as well as a list of multiple IPs allows GYIP to randomly pick an IP from the list and return one of the IPs as a result which simulates DNS round-robin. (Note: even though this is simulates round-robin it does not maintain state and is essentially a random result.) Notice where the `rr` command falls inside the query; right after the domain.
```bash
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.rr.gyip.io @localhost +short A
10.0.0.1
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.rr.gyip.io @localhost +short A
10.0.0.2
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.rr.gyip.io @localhost +short A
10.0.0.2
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.rr.gyip.io @localhost +short A
10.0.0.2
[]$ dig -p 8053 10.0.0.1.10.0.0.2.10.0.0.3.rr.gyip.io @localhost +short A
10.0.0.3
```
In the given example the random nature is revealed.

#### Failure Percent
As a measure to allow for testing failed DNS commands or intermittent failures to resolve GYIP provides a faculty for simulating falure. The command is `fNN` where NN is equal to the percentage of failures that should be experienced. The command `f25` is 25% failures and `f99` is the maximum at 99%. If you need 100% failure then just try and resolve a non-IP query like `fail.gyip.io`.

```bash
[]$ nslookup -port=8053 10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io localhost
Server:		localhost
Address:	::1#8053

** server can\'t find 10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io: NXDOMAIN
[]$ nslookup -port=8053 10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io localhost
Server:		localhost
Address:	::1#8053

Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: 10.0.0.1
Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: 10.0.0.2
Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: 10.0.0.3
Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: ::ffff:10.0.0.1
Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: ::ffff:10.0.0.2
Name:	10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io
Address: ::ffff:10.0.0.3

[]$ nslookup -port=8053 10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io localhost
Server:		localhost
Address:	::1#8053

** server can\'t find 10.0.0.1.10.0.0.2.10.0.0.3.f50.gyip.io: NXDOMAIN
```
