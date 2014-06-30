package ufDidKey

import(
    "fmt"
    "ufCache"
    "ufConfig"
)

var idkey map[uint64] string

func SyncFromCache() error{
	var err error
	if idkey, err = ufCache.DidStringMap(ufConfig.Redis_didkey_hash); nil != err {
		return err
	}
	return nil
}

func Key(did uint64)(key string, ok bool){
	key,ok = idkey[did]
	return key, ok
}

func PrintAll(){
	for d,k :=  range idkey{
		fmt.Printf("%08d:%s", d, k)
	}
}



func Update2Cache(did uint64, key string)error{

	//struct --> map
	idkey[did] = key

	//key --> cache
	ufCache.DidHashSet(ufConfig.Redis_didkey_hash, did, key)
	return nil
}

