package ufPacket

import (
	"fmt"
	"bytes"
	"errors"
	"encoding/binary"
//	"encoding/json"
	"encoding/hex"
	"crypto/aes"
	"crypto/md5"
	"crypto/cipher"
	"ufConfig"
)


type Header struct {
	Ver, Len uint16
	DID uint64
	TS	 uint32
	Sign [ufConfig.Pkt_sign_size]byte
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

//make header without md5 sign
func HeaderCompose(l uint16, did uint64, ts uint32)(hdr_buf []byte, err error){
	var hdr Header
	hdr.Ver = ufConfig.Pkt_prot_ver1
	hdr.Len = l
	hdr.DID = did
	hdr.TS = ts

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, hdr)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
		return nil, err
	}
	return buf.Bytes(), nil
}


//Compose pkt, sign it
func PacketCompose(hdr []byte, jsn_enc []byte, key_pub []byte)(pkt_buf []byte, err error){

	//concate
	pkt_buf = append(hdr, jsn_enc...)

	//sign
	copy(pkt_buf[ufConfig.Pkt_hdr_sign_offset:], key_pub)

	var new_sign = md5.Sum(pkt_buf)
	copy(pkt_buf[ufConfig.Pkt_hdr_sign_offset:], new_sign[:	])

	return pkt_buf, nil
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


func Encypt(in[]byte, key[]byte, iv[]byte)(out []byte, err error){

	//iv := {1,2,3,4,5,... }
	//block, err := aes.NewCipher(key)
	//aes := cipher.NewCBCEncrypter(block, iv)
	//aes.CryptBlocks(out, in)

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
	aes_cipher := cipher.NewCBCEncrypter(block_cipher, iv)

	//padding
	var pad_len = 16-len(in)%16
	var pad = make([]byte, pad_len)
	for i,_ := range pad{
		pad[i]=byte(pad_len)
	}
	in_pad := append(in,pad...)

	fmt.Printf("%s", hex.Dump(in_pad))

	out = make([]byte, len(in_pad))
	aes_cipher.CryptBlocks(out, in_pad)

	return out, nil
}

func test_enc_dec(){
	fmt.Println("===========================")
	e, _ := Encypt([]byte("hellody"), []byte("0123456789abcdef"), []byte("abcdef0123456789"))
	fmt.Println(e)
	d, _ := Decypt(e, []byte("0123456789abcdef"), []byte("abcdef0123456789"))
	fmt.Println(string(d))
	fmt.Println("===========================")
}



