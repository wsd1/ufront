package ufPacket

import (
	"fmt"
	"bytes"
	"errors"
	"encoding/binary"
	"encoding/json"
	"crypto/aes"
	"crypto/cipher"


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

func Decypt(in[]byte, key[]byte, iv[]byte)(out []byte, err error){

	//iv := {1,2,3,4,5,... }
	//block, err := aes.NewCipher(key)
	//aes := cipher.NewCBCEncrypter(block, iv)
	//aes.CryptBlocks(out, in)

	if len(in)%16 != 0{
		return nil, errors.New("Must multiple of 16.")
	}
	if len(key) != 16{
		return nil, errors.New("Key must have length of 16.")
	}
	if len(iv) != 16{
		return nil, errors.New("IV must have length of 16.")
	}

	var block_cipher cipher.Block
	if block_cipher, err = aes.NewCipher(key); err != nil{
		return nil, errors.New("NewCipher err.")
	}

	aes_cipher := cipher.NewCBCDecrypter(block_cipher, iv)

	out = make([]byte, len(in))
	aes_cipher.CryptBlocks(out, in)

	//remove padding
	var padlen = int(out[len(out)-1])
	if padlen > 16{
		return nil, errors.New(fmt.Sprintf("Pad err:%d.",padlen))
	}

	return out[0:len(out) - padlen], nil
}



func json_handle(pkt []byte, hdr *Header){
	
//	var h Header
	var req map[string] interface{}		//detailed: http://stackoverflow.com/questions/24377907/golang-issue-with-accessing-nested-json-array-after-unmarshalling
	json.Unmarshal(pkt[ufConfig.Pkt_hdr_size:], &req)

	//	if "_didkey_set" == req["method"]{	}
//	fmt.Println(i)
//	fmt.Println(i["xxx"])
//	fmt.Println(i["method"])



}
