package ufStat

import (
	"fmt"
)

type stat_t struct{
	warn int
}

var stat = stat_t{0}


func DeviceWarn(ip string, port int, reason string, detail string){
	fmt.Printf("\n[W](%s:%d) %s, %s\n", ip, port, reason, detail)
	stat.warn++
}


func UfrontWarn(reason string, detail string){
	fmt.Printf("\n[W] %s, %s\n", reason, detail)
	stat.warn++
}