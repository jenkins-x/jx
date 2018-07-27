package util

import (
	"log"
	"net"
	"strings"
)

func isValidIpv4(given net.IP) bool {
	var v = true
	if given.To4() == nil {
		v = false
	}

	// Skip loop back
	if given.IsLoopback() {
		v = false
	}

	return v
}

// Returns a list of ip(s)
// Ref: https://gist.github.com/maniankara/f321a15a9bb4c9e4e92b2829d7d2f169
func GetMyIPs() []net.IP {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Println("Error obtaining address from interface: %s continuing...", i)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// Skip the loop back and other garbage
			if !isValidIpv4(ip) {
				continue
			}
			ips = append(ips, ip)
		}
	}
	return ips
}

// increments ip and sees it does not resolve in the network
// until the given count is reached.
// Ensures consequtive count
// @TODO: handle infinite cases + >255
func FindFreeIPs(ip net.IP, count int) []net.IP {
	var ips []net.IP
	ip4 := ip.To4()
	counter := 0
	for {
		ip4[3]++
		dest, err := net.LookupAddr(ip4.String())
		if len(dest) == 0 && err != nil {
			ips = append(ips, ip4.To16())
		} else {
			counter = 0
		}
		counter++
		if counter == count {
			break
		}
	}
	return ips
}

// String representation of FindFreeIPs
func FindFreeIPRange(ip net.IP, count int) string {
	ips := FindFreeIPs(ip, count)
	return ips[0].To4().String() + "-" + string(ips[count-1].To4().String())
}

// Checks if the given string is of ipRange format
// ipRange: x.x.x.x-y.y.y.y
func ValidateIPRange(ipRange string) bool {
	if strings.Contains(ipRange, "-") {
		ips := strings.Split(ipRange, "-")
		for _, ip := range ips {
			_ip := net.ParseIP(ip)
			if _ip == nil {
				return false
			}
		}
	} else {
		return false
	}
	return true
}
