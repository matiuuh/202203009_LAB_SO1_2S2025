


## Estructura del Proyecto

```
└── 📁Proyecto2
    └── 📁bash
        ├── cron_spawn_containers.sh
        ├── install_cron.sh
        ├── load_modules.sh
        ├── remove_cron.sh
    └── 📁dashboard
        └── 📁provisioning
            └── 📁dashboards
                └── 📁json
                    ├── p2-overview.json
                ├── dashboards.yml
            └── 📁datasources
                ├── sqlite.yml
        ├── docker-compose.yml
    └── 📁go-daemon
        └── 📁internal
            └── 📁db
                ├── db.go
            └── 📁decider
                ├── decider.go
            └── 📁proc
                ├── proc.go
        ├── daemon_202203009
        ├── go.mod
        ├── go.sum
        ├── main.go
    └── 📁imagenes
        └── 📁high-cpu
            ├── Dockerfile
        └── 📁high-ram
            ├── Dockerfile
        └── 📁low
            ├── Dockerfile
    └── 📁modulos-kernel
        ├── .continfo_so1_202203009.ko.cmd
        ├── .continfo_so1_202203009.mod.cmd
        ├── .continfo_so1_202203009.mod.o.cmd
        ├── .continfo_so1_202203009.o.cmd
        ├── .Module.symvers.cmd
        ├── .modules.order.cmd
        ├── .sysinfo_so1_202203009.ko.cmd
        ├── .sysinfo_so1_202203009.mod.cmd
        ├── .sysinfo_so1_202203009.mod.o.cmd
        ├── .sysinfo_so1_202203009.o.cmd
        ├── continfo_so1_202203009.c
        ├── continfo_so1_202203009.ko
        ├── continfo_so1_202203009.mod
        ├── continfo_so1_202203009.mod.c
        ├── continfo_so1_202203009.mod.o
        ├── continfo_so1_202203009.o
        ├── Makefile
        ├── Module.symvers
        ├── modules.order
        ├── sysinfo_so1_202203009.c
        ├── sysinfo_so1_202203009.ko
        ├── sysinfo_so1_202203009.mod
        ├── sysinfo_so1_202203009.mod.c
        ├── sysinfo_so1_202203009.mod.o
        ├── sysinfo_so1_202203009.o
    ├── Manual.md
    └── README.md
```