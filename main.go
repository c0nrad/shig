package main

import (
	"flag"
	"fmt"
)

func main() {

	nameServerPtr := flag.String("ns", "8.8.8.8", "nameserver to use")
	qtype := flag.Int("type", 255, "query type")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Missing host")
		fmt.Println("Usage: ./shig <HOSTNAME>")
		flag.PrintDefaults()
		return
	}

	fmt.Println("Host", flag.Arg(0), "NS:", *nameServerPtr, "type:", *qtype)

	r, err := Query(flag.Arg(0), uint16(*qtype), *nameServerPtr)
	if err != nil {
		panic(err)
	}

	var m Message
	m.Deserialize(r)

	fmt.Println(m)

}
