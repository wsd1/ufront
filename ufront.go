package main


import (
    "fmt"
    "net"
    "os"
    "time"
    "ufPacket"

)

type Header struct {
	ver, len 	uint16
	did			[8]uint8
	ts				[4]uint8
	sign			[16]uint8
}

func main() {
	var conn *net.UDPConn
    service := ":1200"
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkError(err)
    conn, err = net.ListenUDP("udp", udpAddr)
    checkError(err)
    for {
        handleClient(conn)
    }
}

func handleClient(conn *net.UDPConn) {
    var buf = make([]byte, 1460)
    var phdr *ufPacket.Header
    n, addr, err := conn.ReadFromUDP(buf[0:])
    if err != nil {
        return
    }
    fmt.Printf("%dbytes:%s\n\n",n, string(buf[:n]))

	//check parse function
	phdr, err = ufPacket.HeaderParse(buf)
    fmt.Printf("%v\n", phdr)

    daytime := time.Now().String()
    conn.WriteToUDP([]byte(daytime), addr)


	var retbuf []byte
	
	//check compose function
	retbuf, err = ufPacket.HeaderCompose(phdr)
    fmt.Printf("%v\n", retbuf)
    
}
func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
        os.Exit(1)
    }
}


