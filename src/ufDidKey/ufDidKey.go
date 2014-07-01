package ufDidKey

import(
    "fmt"
    "crypto/md5"
    "ufCache"
    "ufConfig"
)

type key_ctx_t struct{
	key_pub [ufConfig.Pkt_sign_size]byte
	IV [ufConfig.Pkt_sign_size]byte
	key_priv [ufConfig.Pkt_sign_size]byte
}


var idkey_ctx map[uint64] key_ctx_t

func SyncFromCache() error{
	var err error
	var idkey map[uint64] string
	idkey_ctx = make(map[uint64] key_ctx_t)

	if idkey, err = ufCache.DidStringMap(ufConfig.Redis_didkey_hash); nil != err {
		return err
	}

	for d,k := range idkey{
		idkey_ctx[d] = key2ctx(k)
	}

	return nil
}

func KeyCtxs(did uint64)(key_pub []byte, IV []byte, key_priv []byte, ok bool){
	ctx,ok := idkey_ctx[did]
	if !ok {
		return nil,nil,nil,ok
	}
	return ctx.key_pub[:],ctx.IV[:],ctx.key_priv[:], ok
}

func keyCtx(did uint64)(ctx key_ctx_t, ok bool){
	ctx,ok = idkey_ctx[did]
	return ctx, ok
}


func PrintAll(){
	for d,k := range idkey_ctx{
		fmt.Printf("%08d:%s\n", d, string(k.key_pub[:]))
	}
}

func Update2Cache(did uint64, key string)error{
	//struct --> map
	idkey_ctx[did] = key2ctx(key)

	//key --> cache
	ufCache.DidHashSet(ufConfig.Redis_didkey_hash, did, key)
	return nil
}

func key2ctx(key string) key_ctx_t{
	var ctx key_ctx_t

	copy(ctx.key_pub[0:], key)

	//key = MD5(key_pub)
	var key_priv = md5.Sum(ctx.key_pub[:])
	copy(ctx.key_priv[0:], key_priv[0:])

	//IV = MD5(MD5(key_pub) + key_pub))
	var iv = md5.Sum(append(ctx.key_priv[0:],ctx.key_pub[:]...))
	copy(ctx.IV[0:], iv[0:])

	return ctx
}
