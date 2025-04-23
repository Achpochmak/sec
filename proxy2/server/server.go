package server

import (
	"fmt"
	"net"
)

func Run(port int, handler func(net.Conn) error) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		Port: port,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Listening at port %d...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go func() {
			defer conn.Close()
			err := handler(conn)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
}
