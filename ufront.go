package main


import (
    "fmt"
    "net"
    "os"
    "time"
    "ufPacket"
    "ufCache"
    "ufConfig"
    "ufOL"
)




var idkey map[uint64] string


func main() {

	var err error

	//connect redis
	ufCache.Init();

	if idkey, err = ufCache.DidStringMap(ufConfig.Redis_didkey_hash); nil != err {
		fmt.Println(err)
	}else{
		fmt.Println(idkey)
	}

	ufOL.Sync_from_cache()

	ufOL.Update_to_cache(7542, "192.168.31.7", 635)



	//setup UDP socket 
	var conn *net.UDPConn
    udpAddr, err := net.ResolveUDPAddr("udp4", ":1200")
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

	//recv packet
    n, addr, err := conn.ReadFromUDP(buf[0:])
    if err != nil {
        return
    }
    fmt.Printf("%dbytes:%s\n\n",n, string(buf[:n]))

	//extract header
	phdr, err = ufPacket.HeaderParse(buf)
    fmt.Printf("%v\n", phdr)

	//if err, call sercurity

	//integrity check
	//if err, call sercurity

	//decrypt

	//parse json
	//if err, call sercurity

	//inject to redis

	
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

