package main

import "net"

func main() {
	_, err := net.Dial("tcp", "localhost:3000")
	if err != nil {
		println(err)
	}
}
