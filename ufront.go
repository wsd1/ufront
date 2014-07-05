package main


import (
	"fmt"
	"net"
	"os"
	"time"
	"bytes"
	"errors"
	"ufPacket"
	"ufCache"
	"ufDidKey"
	"ufOL"
	"ufStat"
	"ufConfig"
	"ufSync"

	"crypto/md5"

	"encoding/json"
	"encoding/hex"
)

/*
Principle:
	wait 3 seconds, re-send only one time.

servce:
1. Up req
	a. 1st req or notify: caching
	b. re-req: Find key expire & ack, or skip it.

2. Up ack
	Sending & set key expire 3s

3. Down req
	Sending & record@TimeWait local map & set key expire 3+1s
	thread: check TimeWait local map, if expire and cached not expire, resend.
4. Down ack
	remove from local map & cache ack

*/

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
//		fmt.Println("live...\n")

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


	//recv packet
	pkt_len, addr, err := conn.ReadFromUDP(pkt_buf[0:])
	if err != nil {
		return
	}
	//fmt.Printf("[I]In(%dB)",pkt_len) //, string(pkt_buf[:pkt_len])

	jsn_ele, err := up_pkt_handle(pkt_buf[:pkt_len], addr.IP.String(), addr.Port)
	if nil != err {
		return
	}


	var did uint64
	did, ok := jsn_ele[ufConfig.JSON_TAG_EX_DID].(uint64)
	if !ok {
		fmt.Println("Impossible!!!!")
		return
	}

	var method_id int
	if nil != jsn_ele[ufConfig.JSON_TAG_id]{
		method_id = int(jsn_ele[ufConfig.JSON_TAG_id].(float64))
	}else{
		method_id = 0
	}



	switch {
		case nil != jsn_ele[ufConfig.JSON_TAG_method] && nil != jsn_ele[ufConfig.JSON_TAG_params]:		//uplink request


			fmt.Printf("UpReq,method:%s \n", jsn_ele[ufConfig.JSON_TAG_method])

			jsn_bytes, err := json.Marshal(jsn_ele)
			if err != nil {
				ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_JsonParse, fmt.Sprintf("Marshal err:%v",jsn_ele))
			}

			//inject to redis
			ufCache.Transact_push(string(jsn_bytes))

			if 0 != method_id{
				//do sth

			}

		case nil != jsn_ele[ufConfig.JSON_TAG_result]:	//downlink ack, ok
			fmt.Printf("DnAck\n")

		case nil != jsn_ele[ufConfig.JSON_TAG_error]:	//downlink ack, err
			fmt.Printf("Dnlink ack, err\n")

		default:
			ufStat.Warn(addr.IP.String(), addr.Port, ufConfig.ERR_JsonRPC, fmt.Sprintf("DID: %d", did))
			return
	}

	//update cache
	ufOL.Update2Cache(did, addr.IP.String(), addr.Port)


/*
	var retbuf []byte	
	//check compose function
	retbuf, err = ufPacket.HeaderCompose(phdr)
	fmt.Printf("%v\n", retbuf)
*/
	

}


func up_pkt_handle(pkt_buf []byte, IP string, port int)(jsn map[string] interface{}, err error){

	var phdr *ufPacket.Header
	var pkt_len = len(pkt_buf)
	var err_info string
	//if err, call sercurity
	if pkt_len < int(ufConfig.Pkt_hdr_size){
		err_info = "pkt too short"
		ufStat.Warn(IP, port, ufConfig.ERR_PacketHeader, err_info)
		return nil, errors.New(err_info)
	}

	//extract header
	phdr, err = ufPacket.HeaderParse(pkt_buf)

	//if err, call sercurity
	if nil != err{
		err_info = fmt.Sprintf("Pkt Hdr:%s", err)
		ufStat.Warn(IP, port, ufConfig.ERR_PacketHeader, err_info)
		return nil, errors.New(err_info)
	}

	//time stamp error
	var delta = int64(phdr.TS) - int64(ufSync.TS())
//	fmt.Printf("[TS:%d,(%d)]",phdr.TS, delta)
	fmt.Printf("[I]DID:%d,len:%dB,Δ:%ds ", phdr.DID, phdr.Len, delta)
	if delta < 0{delta = -delta}
	if delta > 60 {
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.Warn(IP, port, ufConfig.ERR_SyncErr, err_info)
		return nil, errors.New(err_info)
	}

	//require key
	key_pub, iv, key_priv, ok := ufDidKey.KeyCtxs(phdr.DID);
	if !ok {
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.Warn(IP, port, ufConfig.ERR_KeyMissing, err_info)
		return nil, errors.New(err_info)
	}

	//integrity check

//	fmt.Printf("\npbuf before pad: \n%s\n", hex.Dump(pkt_buf[:pkt_len]))
	copy(pkt_buf[ufConfig.Pkt_hdr_sign_offset:], key_pub)
//	fmt.Printf("\npbuf after pad: \n%s\n", hex.Dump(pkt_buf[:pkt_len]))

	var new_sign = md5.Sum(pkt_buf)
//	fmt.Printf("MD5 result: \n%s\n", hex.EncodeToString(new_sign[:]))


	//if err, call sercurity
	if !bytes.Equal(new_sign[:], phdr.Sign[:]){
		err_info = fmt.Sprintf("DID: %d origin: %s,calculated: %s.", phdr.DID, hex.EncodeToString(phdr.Sign[:]), hex.EncodeToString(new_sign[:]))
		ufStat.Warn(IP, port, ufConfig.ERR_Integrity, err_info)
		return nil, errors.New(err_info)
	}else{
		fmt.Printf("->md5[ok]")
	}

	//decrypt
	pkt_jsn, err := ufPacket.Decypt(pkt_buf[ufConfig.Pkt_hdr_size:pkt_len], key_priv, iv)
	if nil != err{
		err_info = fmt.Sprintf("DID:%d,%s", phdr.DID, err)
		ufStat.Warn(IP, port, ufConfig.ERR_Decrypt, err_info)
		fmt.Printf("\nDump:\n%s\n", hex.Dump(pkt_buf[ufConfig.Pkt_hdr_size:pkt_len]))
		return nil, errors.New(err_info)
	}

	fmt.Printf("->dec[ok] %dB\n", len(pkt_jsn))

	//parse json
	var jsn_ele map[string] interface{}
	if err = json.Unmarshal(pkt_jsn, &jsn_ele); nil != err{
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.Warn(IP, port, ufConfig.ERR_JsonParse, err_info)
		fmt.Printf("\nDecrypt dump:\n%s\n", hex.Dump(pkt_jsn))
		return nil, errors.New(err_info)
	}

	jsn_ele["EX_TS"] = phdr.TS
	jsn_ele["EX_DID"] = phdr.DID
	jsn_ele["EX_IP"] = IP
	jsn_ele["EX_PORT"] = port

	return jsn_ele, nil
}


func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}

