package ufCache

import (
	"time"
	"fmt"
	"strings"
	"ufConfig"
	"github.com/garyburd/redigo/redis"
)

var pool *redis.Pool = nil

func Init(){
	pool = &redis.Pool{  
        MaxIdle: 3,  
        IdleTimeout: 240 * time.Second,  
        Dial: func () (redis.Conn, error) {  
            c, err := redis.Dial("tcp", ufConfig.Redis_server)  
            if err != nil {
                return nil, err
            }
            if _, err := c.Do("AUTH", ufConfig.Redis_password); err != nil {
                c.Close()
                return nil, err
            }
            return c, err
        },  
        TestOnBorrow: func(c redis.Conn, t time.Time) error {  
            _, err := c.Do("PING")  
            return err  
        },  
    }  
}

//list all keys
func hkeysU64(hash string) (keys []uint64, err error){

    c := pool.Get()
    defer c.Close()

	var res []uint64

    values, err := redis.Values(c.Do("HKEYS", hash))
    if err != nil {
        fmt.Println(err)
        return nil, err
    }
    //else{		fmt.Printf("%#v\n", values)	}

	if err = redis.ScanSlice(values, &res); err != nil {
		return nil, err
	}

	//fmt.Printf("%#v\n", res)
	return res, nil
}


func DidStringMap(hash string)(m map[uint64]string, err error) {

	m = make(map[uint64]string)

	//all keys
	keys, err := hkeysU64(hash)
	if nil != err {
		fmt.Println(err)
		return nil, err;
	}

    c := pool.Get()
    defer c.Close()
    var kval string
	for _, k := range keys{
	    kval, err = redis.String(c.Do("HGET", hash, k))
	    if err != nil {
	        fmt.Println(err)
	        return nil, err
	    }
	    m[k] = kval
	}
	return m, nil
}



func DidHashSet(hash string, did uint64, str string)error {

    c := pool.Get()
    defer c.Close()

	if _, err := c.Do("HSET", hash, did, str); nil != err{
		return err	
	}
	return nil
}


func DidHashDel(hash string, did uint64)error {

    c := pool.Get()
    defer c.Close()

	if _, err := c.Do("HDEL", hash, did); nil != err{
		return err	
	}
	return nil
}


//Set time wait buf
func TimeWaitInsert(prefix string, did uint64, method_id int64, enc_jsn []byte) error{
    c := pool.Get()
    defer c.Close()

    var triple = make([]string,3)
    triple[0] = prefix
    triple[1] = fmt.Sprintf("%d", did)
    triple[2] = fmt.Sprintf("%d", method_id)
    h := strings.Join(triple, ":")
    //fmt.Println(h)

	if _, err := c.Do("SETEX", h, ufConfig.Time_wait_sec+1, enc_jsn); nil != err{
		return err	
	}
	return nil
}


func TimeWait(prefix string, did uint64, method_id int64)(enc_jsn []byte, err error){
    c := pool.Get()
    defer c.Close()

    var triple = make([]string,3)
    triple[0] = prefix
    triple[1] = fmt.Sprintf("%d", did)
    triple[2] = fmt.Sprintf("%d", method_id)
    h := strings.Join(triple, ":")
	//fmt.Println(h)

    val, err := redis.Bytes(c.Do("GET", h))
	return val, err
}


func ListPush(list string, val string) error{
    c := pool.Get()
    defer c.Close()

	if _, err := c.Do("LPUSH", list, val); nil != err{
		return err	
	}
	return nil
}

func ListPop(list string)(val string, err error){
    c := pool.Get()
    defer c.Close()
    val, err = redis.String(c.Do("RPOP", list))
	return val, err
}



