obj-m += sysinfo_so1_202203009.o
obj-m += continfo_so1_202203009.o

all:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules
clean:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean