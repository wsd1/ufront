package ufPacket

import (
	"fmt"
	"bytes"
	"encoding/binary"
)

type Header struct {
	Ver, Len 	uint16
	DID			uint64
	TS				uint32
	Sign			[16]uint8
}

func HeaderParse(pkt []byte)(phdr *Header, err error){
	var ufh Header

	fmt.Println("HeaderParse start")

	buf := bytes.NewReader(pkt)
	err = binary.Read(buf, binary.BigEndian, &ufh)

	if err != nil {
		fmt.Println("HeaderParse failed:", err)
		return nil, err
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

