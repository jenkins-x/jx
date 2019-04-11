package main

import (
	"github.com/sirupsen/logrus"
	"net"
)

func main() {
	//host := "a79adab7027a111e8b6e502c017ed5f7-1934607766.eu-west-1.elb.amazonaws.com"
	host := "google.com"

	ips, err := net.LookupIP(host)
	if err != nil {
		logrus.Infof("Failed: %s\n", err)
		return
	}

	for _, ip := range ips {
		logrus.Infof("IP: %s\n", ip.String())
	}

	logrus.Info("Done!")
}
