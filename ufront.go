package main


import (
	"fmt"
	"net"
	"os"
	"time"
	"bytes"
	"ufPacket"
	"ufCache"
	"ufDidKey"
	"ufOL"
	"ufStat"
	"ufConfig"
	"ufSync"

	"crypto/md5"
	"crypto/aes"
	"crypto/cipher"

	"encoding/json"
	"encoding/hex"
)


func main() {

	var err error

	//connect redis
	ufCache.Init();

	ufDidKey.SyncFromCache()
	ufOL.SyncFromCache()

	if err = ufSync.SntpSync(); nil != err{
		fmt.Println(err)
		return
	}

	ufDidKey.PrintAll()

	ufOL.Update2Cache(7542, "192.168.31.7", 635)


	//setup UDP socket 
	var conn *net.UDPConn
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", ufConfig.Server_port))
	checkError(err)
	conn, err = net.ListenUDP("udp", udpAddr)
	checkError(err)

	go uplink_routine(conn)

	for{
		time.Sleep(10000000000)
		fmt.Println("live...\n")

//		daytime := time.Now().String()
//		conn.WriteToUDP([]byte(daytime), addr)

	}

}
func uplink_routine(conn *net.UDPConn){
	for {
		handleClient(conn)
	}
}

func handleClient(conn *net.UDPConn) {
	var pkt_buf = make([]byte, 1460)
	var phdr *ufPacket.Header

	//recv packet
	pkt_len, addr, err := conn.ReadFromUDP(pkt_buf[0:])
	if err != nil {
		return
	}
	fmt.Printf("[I]In %dB",pkt_len) //, string(pkt_buf[:pkt_len])

	//if err, call sercurity
	if pkt_len < int(ufConfig.Pkt_hdr_size){
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_PacketHeader, "pkt too short")
		return
	}

	//extract header
	phdr, err = ufPacket.HeaderParse(pkt_buf)

	//if err, call sercurity
	if nil != err{
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_PacketHeader, fmt.Sprintf("When parse, find %s", err))
		return
	}

	//time stamp error
	var delta = int64(phdr.TS) - int64(ufSync.TS())
//	fmt.Printf("[TS:%d,(%d)]",phdr.TS, delta)
	fmt.Printf("[DID:%d](%dbyts %ds)", phdr.DID, phdr.Len, delta)
	if delta < 0{delta = -delta}
	if delta > 60 {
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_SyncErr, fmt.Sprintf("DID: %d", phdr.DID))
		return
	}

	//require key
	key, ok := ufDidKey.Key(phdr.DID);
	if !ok {
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_KeyMissing, fmt.Sprintf("DID: %d", phdr.DID))
		return
	}

	//integrity check

	//Prepare key[] slice
	var bkey [ufPacket.SignLen]byte
	copy(bkey[0:], key)

//	fmt.Printf("\npbuf before pad: \n%s\n", hex.Dump(pkt_buf[:pkt_len]))
	copy(pkt_buf[ufConfig.Pkt_hdr_sign_offset:], bkey[0:])
//	fmt.Printf("\npbuf after pad: \n%s\n", hex.Dump(pkt_buf[:pkt_len]))

	new_sign := md5.Sum(pkt_buf[:pkt_len])
//	fmt.Printf("MD5 result: \n%s\n", hex.EncodeToString(new_sign[:]))


	//if err, call sercurity
	if !bytes.Equal(new_sign[:], phdr.Sign[:]){
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_Integrity, fmt.Sprintf("DID: %d origin: %s,calculated: %s.", phdr.DID, hex.EncodeToString(phdr.Sign[:]), hex.EncodeToString(new_sign[:])))
		return
	}else{
		fmt.Printf("->md5[ok]")
	}


	//decrypt
	//http://www.oschina.net/code/snippet_197499_25891
	//https://gist.github.com/temoto/5052503
	//http://golang-examples.tumblr.com/post/41866136734/aes-encryption-in-cbc-mode
	//http://blog.studygolang.com/2013/01/go%E5%8A%A0%E5%AF%86%E8%A7%A3%E5%AF%86%E4%B9%8Baes/

	var cip cipher.Block
	if cip, err = aes.NewCipher(bkey[:]); err != nil{
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_Decrypt, fmt.Sprintf("DID: %d", phdr.DID))
		return
    }else{
		fmt.Printf("->decrypting")
	}

	cip.Decrypt(pkt_buf[ufConfig.Pkt_hdr_size:], pkt_buf[ufConfig.Pkt_hdr_size:pkt_len])

	//parse json
	var jsn_ele map[string] interface{}
	if err = json.Unmarshal(pkt_buf[ufConfig.Pkt_hdr_size:], &jsn_ele); nil != err{
		ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_JsonParse, fmt.Sprintf("DID: %d", phdr.DID))
		return
	}else{
		fmt.Printf("->json ok")
	}

	switch {
		case nil != jsn_ele["method"] && nil != jsn_ele["params"]:		//uplink request
			fmt.Printf("Uplink request, method:%s \n", jsn_ele["method"])

			//inject to redis

		case nil != jsn_ele["result"]:	//downlink ack, ok
			fmt.Printf("Downlink ack, ok\n")

		case nil != jsn_ele["error"]:	//downlink ack, err
			fmt.Printf("Downlink ack, err\n")

		case true:
			ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_JsonRPC, fmt.Sprintf("DID: %d", phdr.DID))
			return
	}


	//update cache
	ufOL.Update2Cache(phdr.DID, addr.IP.String(), addr.Port)


/*
	var retbuf []byte	
	//check compose function
	retbuf, err = ufPacket.HeaderCompose(phdr)
	fmt.Printf("%v\n", retbuf)
*/
	

}



func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}

