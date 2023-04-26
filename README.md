# a_uintptr_bug
 一个uintptr触发的bug

**这个事情是上周在业务代码里发生的，由于代码涉及到公司的商业机密，所以抽象出来描述一下**

非常好，100%复现bug，当fmt.Println语句去掉以后，函数每次调用都产生的不同的数据，当fmt.Println加上以后，返回值又变正确了

简直充满了奇异的量子感“只要不观察过程，返回值就是随机的”，不由得怀疑自己可能买了一台量子计算机

拉了平台组大家一起来围观了这个神奇的黑魔法bug以后，在上面加一句注释：“不要动，我也不知道为什么，这是黑魔法，无法理解的，注释掉这个函数就崩了，要用这句print来保证稳定运作”……好的，bug解决了，下班咯~
……
……
……
……
……才怪哦

正经的高级工程师肯定不能干这种事啊……代码一定是讲道理的，不可能有任何无法预测的行为发生，无法预测的行为一定代表了代码中存在某些逻辑错误

采用go工具检查编译条件，发现print行为本身造成了一个本来位于栈上的对象被挪到堆上

```

./main.go:70:6: can inline StructToByte
./main.go:49:16: inlining call to fmt.Println
./main.go:51:27: inlining call to StructToByte
./main.go:67:16: inlining call to fmt.Println
./main.go:30:15: minBody does not escape
./main.go:31:20: make([]byte, 0) escapes to heap
./main.go:32:17: &PackageDataHead{...} does not escape
./main.go:38:18: &PackageCheckHead{} does not escape
./main.go:41:20: &PackageHead{...} does not escape
./main.go:45:16: &MessagePackage{...} escapes to heap
./main.go:49:16: ... argument does not escape
./main.go:51:27: &reflect.SliceHeader{...} does not escape
./main.go:65:20: make([]byte, 0) does not escape
./main.go:67:16: ... argument does not escape
./main.go:67:17: "最终返回值------>" escapes to heap
./main.go:67:17: res escapes to heap
./main.go:70:19: structData does not escape
./main.go:72:19: &reflect.SliceHeader{...} does not escape
```
注意./main.go:45:16:这句，当不带print的时候进行分析，可以发现这个MessagePackage本来应该是一个栈内存对象

```

./main.go:70:6: can inline StructToByte
./main.go:51:27: inlining call to StructToByte
./main.go:67:16: inlining call to fmt.Println
./main.go:30:15: minBody does not escape
./main.go:31:20: make([]byte, 0) escapes to heap
./main.go:32:17: &PackageDataHead{...} does not escape
./main.go:38:18: &PackageCheckHead{} does not escape
./main.go:41:20: &PackageHead{...} does not escape
./main.go:45:16: &MessagePackage{...} does not escape
./main.go:51:27: &reflect.SliceHeader{...} does not escape
./main.go:65:20: make([]byte, 0) does not escape
./main.go:67:16: ... argument does not escape
./main.go:67:17: "最终返回值------>" escapes to heap
./main.go:67:17: res escapes to heap
./main.go:70:19: structData does not escape
./main.go:72:19: &reflect.SliceHeader{...} does not escape
```
进一步查看汇编，发现一个有意思的指令

```asm
0x00e3 00227 (main.go:45)    LEAQ    type.main.MessagePackage(SB), AX
0x00ea 00234 (main.go:45)    CALL    runtime.newobject(SB)
0x00ef 00239 (main.go:45)    MOVQ    AX, main.message+128(SP)
0x00f7 00247 (main.go:46)    MOVWLZX    main..autotmp_27+82(SP), DX
0x00fc 00252 (main.go:46)    MOVBLZX    main..autotmp_27+84(SP), SI
```

MessagePackage对象被放在SB寄存器以后，有print的情况下，会额外new一次SB寄存器
那么为什么new了以后理论上应该是一个bug的行为却反而修好了bug呢？

继续看源代码，发现结构体转换到字节中，用uintptr操作unsafe.Pointer指针

但是官方文档中，uintptr并不被视为指针，而是一个单纯的uint64数据用来描述“指针转成uint64以后是多少”，这个数据应该是用来推导连续数据结构的，而不应该用来取值，因此当用uint64指针记录对象位置以后，一旦对象位置发生变化（比如数组append行为导致重新分配地址），uintptr不会发生变化，此时继续调用uintptr进行取值后获取到的就已经不再是原来的值了

当一个函数的栈的大小改变时，一个新的内存段将申请给此栈使用。原先已经开辟在老的内存段上的内存块直接就被转移到新的内存段上，同时，引用着这些开辟在此栈上的内存块的指针以及指针中存储的地址也将得到刷新，但是！uintptr却依旧指向原来的位置！

所以

```go
r := unsafe.Pointer(headerData)
buffData := *(*[]byte)(r)
```
这两句执行的时候，由于headerData中保存的是uintptr，于是自然而然就被指向全新的位置了

那么答案就呼之欲出了

当fmt诱发堆逃逸以后，由于go的堆逃逸存在如下情况

如果一个结构体值的一个字段逃逸到了堆上，则此整个结构体值也逃逸到了堆上
如果一个数组的某个元素逃逸到了堆上，则此整个数组也逃逸到了堆上。
如果一个切片的某个元素逃逸到了堆上，则此切片中的所有元素都将逃逸到堆上，但此切片值的直接部分(SliceHeader)可能开辟在栈上。
如果一个值部v被一个逃逸到了堆上的值部所引用，则此值部v也将逃逸到堆上
于是，fmt.Println把结构体给逃逸到堆上了，导致整个结构体都逃逸了，进而接下来栈内存的重新分配已经和这个“虽然依旧是函数内部变量”的结构体毫无关系，反而保证了uintptr取值在被释放之前都不会发生变化，然后go又不会轻易释放堆内存……于是内存逃逸/泄露反而保证了数据的一致性
