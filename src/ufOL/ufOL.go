package ufOL

import(
    "fmt"
    "encoding/json"
    "ufCache"
    "ufConfig"
    "ufSync"
)


type ol_info struct{
	IP string	`json:"ip"` 
	Port int	`json:"port"` 
	IP_last string	`json:"ip_last"` 
	Port_last int	`json:"port_last"` 
	Timestamp uint32	`json:"ts"` 
}
//hset ufOnline 234 '{"ip":"192.168.45.78", "port":12345, "ip_last":"354.254.125.32", "port_last":54632, "ts":123456}'

var online_sock map[uint64] ol_info


func SyncFromCache()(err error){
	online_sock, err = get_from_cache()
	
	fmt.Println(online_sock)
	return err
}


func Info(did uint64)(ip string, port int, ip_last string, port_last int, ts uint32, ok bool){
	if i, ok := online_sock[did]; ok{
		return i.IP, i.Port, i.IP_last, i.Port_last, i.Timestamp, ok
	}

	return "", 0, "", 0, 0, false
}


func Update2Cache(did uint64, ip string, port int)(elapse int, err error){

	var sck = ol_info{}
	sck.IP = ip
	sck.Port = port
	sck.Timestamp = ufSync.TS()

	//update sck if exsit
	if ip_, port_, _, _, ts_, ok := Info(did); ok{
		sck.IP_last = ip_
		sck.Port_last = port_
		elapse = int(sck.Timestamp) - int(ts_)
	}else{
		sck.IP_last = ""
		sck.Port_last = 0
		elapse = 0
	}

		
	//struct --> json
	jsn, err := json.Marshal(&sck)
	if nil != err{
		return 0, err
	}

	//struct --> map
	online_sock[did] = sck

	//json --> cache
	ufCache.DidHashSet(ufConfig.Redis_olinfo_hash, did, string(jsn))

	return elapse, nil
}



func get_from_cache()(ret_info map[uint64] ol_info, err error){

	ret_info = make(map[uint64] ol_info)

	//get did -> json
	var info map[uint64] string

	if info, err = ufCache.DidStringMap(ufConfig.Redis_olinfo_hash); nil != err {
		fmt.Println(err)
		return nil, err
	}

	//convert json -> struct
	for did, jsn := range info {
		var ol ol_info

		err = json.Unmarshal([]byte(jsn), &ol)
		if nil != err{
			break;
			return nil, err;
		}

		ret_info[did] = ol
	}
	return ret_info, nil
}



