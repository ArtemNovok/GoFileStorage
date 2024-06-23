package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:3000")
	if err != nil {
		println(err)
	}
	conn.Write([]byte("hi"))
	fmt.Println("sendes message 1")
	time.Sleep(5 * time.Second)
	conn.Write([]byte("from peer"))
	fmt.Println("sendes message 2")
	select {}
}
