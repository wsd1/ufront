package main

import (
	"fmt"
	"time"
	"ufront"
//	"ufPacket"
	"ufCache"
	"ufDidKey"
	"ufOL"
//	"ufStat"
	"ufConfig"
	"ufSync"
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

	//ufDidKey.PrintAll()

	//setup UDP socket 
	ufront.Init()



	go uplink_routine()
	go dnlink_routine()

	for{
		time.Sleep(10000000000)
//		fmt.Println("live...\n")

//		daytime := time.Now().String()
//		conn.WriteToUDP([]byte(daytime), addr)
	}
}


func dnlink_routine(){
	for{
		jsn, err := ufCache.ListPop(ufConfig.Redis_dn_req_list)
		if err == nil {
			ufront.Dnlink_msg_handle(jsn)
		}

		jsn, err = ufCache.ListPop(ufConfig.Redis_up_ack_list)
		if err == nil {
			ufront.Dnlink_msg_handle(jsn)
		}


	}


}

func uplink_routine(){
	var pkt_buf = make([]byte, 1460)

	for {
		//recv packet
		pkt_len, sip, sport, err := ufront.Pkt_read(pkt_buf)
		if err != nil {
			continue
		}
		//fmt.Printf("[I]In(%dB)",pkt_len) //, string(pkt_buf[:pkt_len])
		ufront.Uplink_pkt_handle(pkt_buf, pkt_len, sip, sport)
	}
}
