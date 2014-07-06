package ufOL

import(
    "fmt"
    "encoding/json"
    "ufCache"
    "ufConfig"
    "ufSync"
)


type OL_info struct{
	IP string	`json:"ip"` 
	Port int	`json:"port"` 
	IP_last string	`json:"ip_"` 
	Port_last int	`json:"port_"` 
	Timestamp uint32	`json:"ts"` 
}
//hset ufOnline 234 '{"ip":"192.168.45.78", "port":12345, "ip_last":"354.254.125.32", "port_last":54632, "ts":123456}'

var online_sock map[uint64] OL_info


func SyncFromCache()(err error){
	online_sock, err = get_from_cache()
	//fmt.Println(online_sock)
	return err
}

//Get info from local
func Info(did uint64)(inf OL_info, ok bool){
	i, ok := online_sock[did]
	return i, ok
}


func Update2Cache(did uint64, ip string, port int)(elapse int, err error){
	var sck = OL_info{}
	sck.IP = ip
	sck.Port = port
	sck.Timestamp = ufSync.TS()

	//update sck if exsit
	if i, ok := Info(did); ok{
		sck.IP_last = i.IP
		sck.Port_last = i.Port
		elapse = int(sck.Timestamp) - int(i.Timestamp)
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

	//struct --> local map
	online_sock[did] = sck

	//json --> cache
	ufCache.DidHashSet(ufConfig.Redis_olinfo_hash, did, string(jsn))

	return elapse, nil
}



func get_from_cache()(ret_info map[uint64] OL_info, err error){

	ret_info = make(map[uint64] OL_info)

	//get did -> json
	var info map[uint64] string

	if info, err = ufCache.DidStringMap(ufConfig.Redis_olinfo_hash); nil != err {
		fmt.Println(err)
		return nil, err
	}

	//convert json -> struct
	for did, jsn := range info {
		var ol OL_info

		err = json.Unmarshal([]byte(jsn), &ol)
		if nil != err{
			break;
			return nil, err;
		}

		ret_info[did] = ol
	}
	return ret_info, nil
}



