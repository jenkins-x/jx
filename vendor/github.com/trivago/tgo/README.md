# tGo

[![GoDoc](https://godoc.org/github.com/trivago/tgo?status.svg)](https://godoc.org/github.com/trivago/tgo)
[![Go Report Card](http://goreportcard.com/badge/trivago/tgo)](http://goreportcard.com/report/trivago/tgo)
[![Build Status](https://travis-ci.org/trivago/tgo.svg?branch=master)](https://travis-ci.org/trivago/tgo)

Trivago go extensions and utilities.
This package contains utility functions and structs that aid trivago with golang development across different projects.

This package and all subpackage match the golang standard library package names along with a "t" prefix.
I.e. types that would be placed in the "net" package can be found in the "tnet" package, etc..
This prefix was chosen to allow mixing standard libary and tgo without having to rename package imports all the time.
