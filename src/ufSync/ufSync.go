package ufSync

import(
	"fmt"
	"net"
	"time"
	"ufConfig"
)

var is_late bool
var delta int64

func SntpSync()error{
	
	raddr, err := net.ResolveUDPAddr("udp", ufConfig.SNTP_server)
	if err != nil {
		return err
	}

	data := make([]byte, 48)
	data[0] = 3<<3 | 3

	con, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}

	defer con.Close()

	_, err = con.Write(data)
	if err != nil {
		return err
	}

	con.SetDeadline(time.Now().Add(3 * time.Second))

	_, err = con.Read(data)
	if err != nil {
		return err
	}

	var sec uint64
	sec = uint64(data[43]) | uint64(data[42])<<8 | uint64(data[41])<<16 | uint64(data[40])<<24
	//frac = uint64(data[47]) | uint64(data[46])<<8 | uint64(data[45])<<16 | uint64(data[44])<<24

	sec -= (86400 * (365 * 70 + 17))

	delta = int64(sec) - time.Now().Unix()

	fmt.Printf("Network time: %v\n", sec)
	fmt.Printf("System time: %v\n", time.Now().Unix())
	fmt.Printf("Time delta: %v\n", delta)

	return nil
}

func TS()uint32{
	var t = time.Now().Unix() + delta
	return uint32(t)
}


