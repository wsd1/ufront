package ufront


import (
	"fmt"
	"net"
	"os"
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
var conn *net.UDPConn

var last_mid map[uint64] int64

/*
Principle:
	wait 3 seconds, re-send only one time.

servce:
1. Up req
	a. 1st req or notify: caching	[done]
	b. re-send req: Find timewaited cache & ack, or skip it. [done]
	
	Fill in cache[Redis_up_req_list], with TS,DID,IP,PORT tags [done]

2. Up ack
	APP fill in cache[Redis_up_ack_list], with DID tag 

	Sending 	[done]
	Insert timewait cache [done]

3. Down req
	APP fill in cache[Redis_dn_req_list], with DID tag

	Sending 	[done]


4. Down ack
	Fill in cache[Redis_dn_ack_list], with TS,DID,IP,PORT tags 	[done]
*/


func Init(){
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", ufConfig.Server_port))
	checkError(err)
	conn, err = net.ListenUDP("udp", udpAddr)
	checkError(err)

	last_mid = make(map[uint64] int64)
}

func Dnlink_msg_handle(jsn string){

	//parse json
	var jsn_ele map[string] interface{}
	if err := json.Unmarshal([]byte(jsn), &jsn_ele); nil != err{

		ufStat.UfrontWarn(ufConfig.ERR_JsonParse, "DnLink json parse err")
		fmt.Printf("\nParse dump:\n%s\n", hex.Dump([]byte(jsn)))
		return
	}


	//DID tag check
	did_raw, ok := jsn_ele[ufConfig.JSON_TAG_EX_DID].(float64)
	if !ok {
		ufStat.UfrontWarn(ufConfig.ERR_JsonRPC, fmt.Sprintf("DnLink must have %s tag", ufConfig.JSON_TAG_EX_DID))
		fmt.Printf("\nParse dump:\n%s\n", hex.Dump([]byte(jsn)))
		return
	}
	did := uint64(did_raw)

	//remove unnessary tag
	delete(jsn_ele, ufConfig.JSON_TAG_EX_DID)



	//serialize 
	jsn_buf, err := json.Marshal(jsn_ele)
	if err != nil {
	    fmt.Println("Json Marshal err")
	    return
	}


	//get auth info
	_, iv, key_priv, ok := ufDidKey.KeyCtxs(did);
	if !ok {
		err_info := fmt.Sprintf("DnLink did:%d not find", did)
		ufStat.UfrontWarn(ufConfig.ERR_KeyMissing, err_info)
		return
	}

	//encrypt payload
	jsn_enc, err := ufPacket.Encypt(jsn_buf, key_priv, iv)
	if err != nil {
	    fmt.Printf("Json enc err:%s", err)
	    return
	}




	//two type:
	//1. Down link req
	//2. Up link ack  ----> should be cached for at least 3 seconds

	switch{
		case nil != jsn_ele[ufConfig.JSON_TAG_method] && nil != jsn_ele[ufConfig.JSON_TAG_params]:
			fmt.Println("Dnlink req")

		case nil != jsn_ele[ufConfig.JSON_TAG_result] || nil != jsn_ele[ufConfig.JSON_TAG_error]:
			fmt.Println("Uplink ack")

			//must have method_id tag
			mid_raw, ok := jsn_ele[ufConfig.JSON_TAG_id].(float64)
			if !ok {
				ufStat.UfrontWarn(ufConfig.ERR_JsonRPC, fmt.Sprintf("UpLink ack must have %s tag", ufConfig.JSON_TAG_id))
				fmt.Printf("\nParse dump:\n%s\n", hex.Dump([]byte(jsn)))
				return
			}
			method_id := int64(mid_raw)

			//cache it
			if err := ufCache.TimeWaitInsert(ufConfig.Time_wait_ack_prefix, did, method_id, jsn_enc); nil != err{
				fmt.Printf("Timewait insert err:%s", err)
			}

		default:
			ufStat.UfrontWarn(ufConfig.ERR_JsonRPC, fmt.Sprintf("DnLink must have %s tag", ufConfig.JSON_TAG_EX_DID))
			fmt.Printf("\nParse dump:\n%s\n", hex.Dump([]byte(jsn)))
			return
	}




	pkt_buf, err := Pkt_make(did, jsn_enc)
	if nil != err{
		return
	}

	//send
	err = Pkt_send(did, pkt_buf)
	if err != nil {
	    fmt.Printf("Pkt send err:%s", err)
	    return
	}
}


func Uplink_pkt_handle(pkt_buf []byte, pkt_len int, ip string, port int){

	jsn_ele, err := pkt_parse(pkt_buf[:pkt_len], ip, port)
	if nil != err {
		return
	}

	var did uint64
	did, ok := jsn_ele[ufConfig.JSON_TAG_EX_DID].(uint64)
	if !ok {
		fmt.Println("Impossible!!!!")
		return
	}

	//json serialization
	jsn_bytes, err := json.Marshal(jsn_ele)
	if err != nil {
		ufStat.DeviceWarn(ip, port, ufConfig.ERR_JsonParse, fmt.Sprintf("Marshal err:%v",jsn_ele))
	}

	switch {
		//uplink request
		case nil != jsn_ele[ufConfig.JSON_TAG_method] && nil != jsn_ele[ufConfig.JSON_TAG_params]:


			fmt.Printf("UpReq,method:%s \n", jsn_ele[ufConfig.JSON_TAG_method])

			//if it is method
			if nil != jsn_ele[ufConfig.JSON_TAG_id]{

				method_id := int64(jsn_ele[ufConfig.JSON_TAG_id].(float64))

				fmt.Printf("Method ID:%d\n", method_id)

				if is_resend := is_resend_req(did, method_id); !is_resend{
					fmt.Printf("New call, up cached\n")

					//push to consumer
					ufCache.ListPush(ufConfig.Redis_up_req_list, string(jsn_bytes))

				}else{
					fmt.Printf("Re-sent call, ")

					//if cached not expire, re-send
					if enc_payload, err := ufCache.TimeWait(ufConfig.Time_wait_ack_prefix, did, method_id); nil != err {
						fmt.Printf("ack cached, re-send\n")

						pkt_buf, err := Pkt_make(did, enc_payload)
						if nil != err{
							return
						}

						//send
						err = Pkt_send(did, pkt_buf)
						if err != nil {
						    fmt.Printf("Pkt send err:%s", err)
						    return
						}

					}else{
						fmt.Printf("no ack cached, skip\n")
						return //skip
					}

				}


			// If is notify
			}else{

				fmt.Printf("Notification, up cached\n")

				//json serialization
				jsn_bytes, err := json.Marshal(jsn_ele)
				if err != nil {
					ufStat.DeviceWarn(ip, port, ufConfig.ERR_JsonParse, fmt.Sprintf("Marshal err:%v",jsn_ele))
				}

				//inject to redis, push to consumer
				ufCache.ListPush(ufConfig.Redis_up_req_list, string(jsn_bytes))

			}


		//Downlink ack
		case nil != jsn_ele[ufConfig.JSON_TAG_result] || nil != jsn_ele[ufConfig.JSON_TAG_error]:

			fmt.Printf("Dnlink Ack\n")

			//inject to redis, push to consumer
			ufCache.ListPush(ufConfig.Redis_dn_ack_list, string(jsn_bytes))


		default:
			ufStat.DeviceWarn(ip, port, ufConfig.ERR_JsonRPC, fmt.Sprintf("DID: %d", did))
			return
	}

	//update cache
	ufOL.Update2Cache(did, ip, port)

}



func Pkt_read(pkt_buf []byte)(pkt_len int, sip string, sport int, err error){
	pkt_len, addr, err := conn.ReadFromUDP(pkt_buf)
	return pkt_len, addr.IP.String(), addr.Port, err
}


func Pkt_send(did uint64, pkt_buf []byte) error{
	inf, ok := ufOL.Info(did)
	if ok{
		var addr net.UDPAddr
		addr.IP = net.ParseIP(inf.IP)
		addr.Port = inf.Port
		conn.WriteToUDP(pkt_buf, &addr)
		return nil
	}

	return errors.New("UDP send err.")
}

func pkt_parse(pkt_buf []byte, IP string, port int)(jsn map[string] interface{}, err error){

	var phdr *ufPacket.Header
	var pkt_len = len(pkt_buf)
	var err_info string


	if pkt_len < int(ufConfig.Pkt_hdr_size){
		err_info = "pkt too short"
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_PacketHeader, err_info)
		return nil, errors.New(err_info)
	}

	//extract header
	phdr, err = ufPacket.HeaderParse(pkt_buf)

	//if err, call sercurity
	if nil != err{
		err_info = fmt.Sprintf("Pkt Hdr:%s", err)
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_PacketHeader, err_info)
		return nil, errors.New(err_info)
	}

	//time stamp error
	var delta = int64(phdr.TS) - int64(ufSync.TS())
//	fmt.Printf("[TS:%d,(%d)]",phdr.TS, delta)
	fmt.Printf("[I]DID:%d,len:%dB,Δ:%ds ", phdr.DID, phdr.Len, delta)
	if delta < 0{delta = -delta}
	if delta > 60 {
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_SyncErr, err_info)
		return nil, errors.New(err_info)
	}

	//require key
	key_pub, iv, key_priv, ok := ufDidKey.KeyCtxs(phdr.DID);
	if !ok {
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_KeyMissing, err_info)
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
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_Integrity, err_info)
		return nil, errors.New(err_info)
	}else{
		fmt.Printf("->md5[ok]")
	}


	if pkt_len == int(ufConfig.Pkt_hdr_size){
		fmt.Printf("Just header:\n%s", hex.Dump(pkt_buf))
		return nil, errors.New("Just header.")
	}

	//decrypt
	pkt_jsn, err := ufPacket.Decypt(pkt_buf[ufConfig.Pkt_hdr_size:pkt_len], key_priv, iv)
	if nil != err{
		err_info = fmt.Sprintf("DID:%d,%s", phdr.DID, err)
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_Decrypt, err_info)
		fmt.Printf("\nDump:\n%s\n", hex.Dump(pkt_buf[ufConfig.Pkt_hdr_size:pkt_len]))
		return nil, errors.New(err_info)
	}

	fmt.Printf("->dec[ok] %dB\n", len(pkt_jsn))

	//parse json
	var jsn_ele map[string] interface{}
	if err = json.Unmarshal(pkt_jsn, &jsn_ele); nil != err{
		err_info = fmt.Sprintf("DID: %d", phdr.DID)
		ufStat.DeviceWarn(IP, port, ufConfig.ERR_JsonParse, err_info)
		fmt.Printf("\nDecrypt dump:\n%s\n", hex.Dump(pkt_jsn))
		return nil, errors.New(err_info)
	}

	jsn_ele[ufConfig.JSON_TAG_EX_TS] = phdr.TS
	jsn_ele[ufConfig.JSON_TAG_EX_DID] = phdr.DID
	jsn_ele[ufConfig.JSON_TAG_EX_IP] = IP
	jsn_ele[ufConfig.JSON_TAG_EX_PORT] = port

	return jsn_ele, nil
}


// Make packet with encrypted payload and dedicate DID
// key_pub: optional
func Pkt_make(did uint64, enc_payload []byte)(pkt_buf []byte, err error){

	//get auth info
	key_pub, _, _, ok := ufDidKey.KeyCtxs(did);
	if !ok {
		err_info := fmt.Sprintf("DnLink did:%d not find", did)
		ufStat.UfrontWarn(ufConfig.ERR_KeyMissing, err_info)
		return nil, errors.New(err_info)
	}

	//hdr compose
	hdr_buf, err := ufPacket.HeaderCompose(uint16(ufConfig.Pkt_hdr_size) + uint16(len(enc_payload)), did, ufSync.TS())
	if err != nil {
		err_info := fmt.Sprintf("Header compose err:%s", err)
		return nil, errors.New(err_info)
	}

	//hdr + payload and md5 sign
	pkt_buf, _ = ufPacket.PacketCompose(hdr_buf, enc_payload, key_pub)

	return pkt_buf, nil
}


func is_resend_req(did uint64, mid int64)bool{

	last_id, ok := last_mid[did]

	if !ok || mid != last_id{
		last_mid[did] = mid
		return false
	}
	return true
}


func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}

