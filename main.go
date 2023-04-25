package main

import (
	"fmt"
	"log"
	"reflect"
	"time"
	"unsafe"
)

type PackageDataHead struct {
	Head1 uint8
	Head2 uint16
	Head3 uint8
	Head4 uint16
}

type PackageCheckHead struct {
	Check1 uint32
	Check2 uint32
}

type PackageHead struct {
	DataHead  PackageDataHead
	CheckHead PackageCheckHead
}

type MessagePackage struct {
	Head PackageHead
}

func makeData(minBody []byte) []byte {
	msgBuff := make([]byte, 0)
	dataHead := &PackageDataHead{
		Head1: 0x53,
		Head2: uint16(123),
		Head3: 0x53,
		Head4: uint16(123),
	}
	//fmt.Println(dataHead)
	checkHead := &PackageCheckHead{}
	//fmt.Println(checkHead)
	checkHead.Check1 = uint32(655)
	checkHead.Check2 = uint32(655)
	//fmt.Println(checkHead)
	packageHead := &PackageHead{
		DataHead:  *dataHead,
		CheckHead: *checkHead,
	}
	message := &MessagePackage{
		Head: *packageHead,
	}
	log.Println(message)
	//如果传入的minbody是个空的map，千万不要注释掉这句Println
	//这句Println一旦被注释掉，这个函数的返回值就会变成一个随机的字节串
	//只有这句Println开着的时候这个函数的返回值才是正确的
	//可以用fmt.Println代替
	//println不能代替Println，如果这里用println依旧返回值会变成随机字节
	//这是个黑魔法，我也不知道为什么，但是确实不能注释掉
	msgBuff = StructToByte(message)
	result := append(msgBuff, minBody...)
	return result
}

func StructToByte(structData *MessagePackage) []byte {
	msgLen := unsafe.Sizeof(*structData)
	headerData := &reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(structData)),
		Len:  int(msgLen),
		Cap:  int(msgLen),
	}
	msgBuff := *(*[]byte)(unsafe.Pointer(headerData))
	return msgBuff
}

func main() {
	for {
		minBody := make([]byte, 0)
		res := makeData(minBody)
		fmt.Println("最终返回值------>", res)
		time.Sleep(1 * time.Second)
		//break
	}

}
