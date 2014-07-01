package ufPacket

import (
	"fmt"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"unsafe"
	"errors"
	"ufConfig"
)


type Header struct {
	Ver, Len uint16
	DID uint64
	TS	 uint32
	Sign [ufConfig.Pkt_sign_size]uint8
}

func HeaderParse(pkt []byte)(phdr *Header, err error){
	var ufh Header

	buf := bytes.NewReader(pkt)
	err = binary.Read(buf, binary.BigEndian, &ufh)

	if err != nil {
		fmt.Println("HeaderParse failed:", err)
		return nil, err
	}

	if ufh.Ver != ufConfig.Pkt_prot_ver1{
		return nil, errors.New("proto version error")
	}

	return &ufh, nil
}

func HeaderCompose(phdr *Header)(pkt []byte, err error){
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, *phdr)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func md5_check(pkt []byte) bool{
	return true
}

func decypt(pkt []byte){
}

func json_handle(pkt []byte, hdr *Header){
	
	var h Header
	var req map[string] interface{}		//detailed: http://stackoverflow.com/questions/24377907/golang-issue-with-accessing-nested-json-array-after-unmarshalling
	json.Unmarshal(pkt[unsafe.Sizeof(h):], &req)

	//	if "_didkey_set" == req["method"]{	}
//	fmt.Println(i)
//	fmt.Println(i["xxx"])
//	fmt.Println(i["method"])



}
