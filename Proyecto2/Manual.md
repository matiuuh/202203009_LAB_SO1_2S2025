# Manual Tecnico
# Labortario de Sistemas Operativos 1
## Proyecto 2
### Segundo Semestre 2025

```js
Universidad San Carlos de Guatemala

Programador:  Mateo Estuardo Diego Noriega
Carnet:       202203009
```

---

Para desarrollar el proyecto se realizara de la siguiente manera

## Modulos de kernel
### 1. Preparación del entorno
Es importante verificar primero el security bot, este debe de estar deshabilitado para evitar problemas.

De la misma manera debe de instalarse un compilador de kernel como `gcc` por ejemplo. Asimismo, tener acceso a los encabezados del kernel. Si se trabaja en ubuntu estos pueden instalarse por medio del siguiente comando:

```bash
sudo apt-get install gcc linux-headers-$(uname -r)
```

Una vez hecho esto, se debera crear el archivo del modulo.

### 2. Crear el archivo del modulo de kernel
Para el desarrollo de este proyecto se desarrollaron dos modulos de kernel, por lo tanto, a continuacion, se presentan cada uno de los modulos.

#### sysinfo_so1_202203009
Dicho modulo se encarga de de mostrar todos los procesos del sistema, asi como su informacion en formato json, esa muestra el PID, nombre, linea de comando que se ejecuto, vsz, rss, porcentaje de memoria usada, porcentaje de cpu y estado.
```bash
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/string.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/jiffies.h>
#include <linux/uaccess.h>
#include <linux/sched/signal.h>
#include <linux/slab.h>
#include <linux/sched/mm.h>
#include <linux/compiler.h>    /* READ_ONCE */

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Mateo Estuardo Diego Noriega");
MODULE_DESCRIPTION("Modulo para leer informacion de memoria y CPU en JSON (procesos del sistema)");
MODULE_VERSION("1.0");

#define PROC_NAME "sysinfo_so1_202203009"
#define MAX_CMDLINE_LENGTH 256

/* ---- Helpers de estado (kernels 6.x: usar __state) ---- */
static const char *task_state_short(struct task_struct *t)
{
    unsigned int st = READ_ONCE(t->__state);

    if (st == TASK_RUNNING)             return "R"; /* Running */
    if (st & TASK_INTERRUPTIBLE)        return "S"; /* Sleeping (interruptible) */
    if (st & TASK_UNINTERRUPTIBLE)      return "D"; /* Sleeping (uninterruptible) */
#ifdef __TASK_STOPPED
    if (st & __TASK_STOPPED)            return "T"; /* Stopped */
#endif
#ifdef __TASK_TRACED
    if (st & __TASK_TRACED)             return "t"; /* Traced */
#endif
#ifdef TASK_WAKEKILL
    if (st & TASK_WAKEKILL)             return "K"; /* Waking/killable */
#endif
#ifdef TASK_PARKED
    if (st & TASK_PARKED)               return "P"; /* Parked */
#endif
#ifdef TASK_DEAD
    if (st & TASK_DEAD)                 return "X"; /* Dead */
#endif
#ifdef EXIT_ZOMBIE
    if (t->exit_state & EXIT_ZOMBIE)    return "Z"; /* Zombie */
#endif
    return "?";
}

/* Cmdline del proceso (caller debe kfree) */
static char *get_process_cmdline(struct task_struct *task)
{
    struct mm_struct *mm;
    char *cmdline, *p;
    unsigned long arg_start, arg_end;
    int i, len;

    cmdline = kmalloc(MAX_CMDLINE_LENGTH, GFP_KERNEL);
    if (!cmdline)
        return NULL;

    mm = get_task_mm(task);
    if (!mm) {
        kfree(cmdline);
        return NULL;
    }

    down_read(&mm->mmap_lock);
    arg_start = mm->arg_start;
    arg_end   = mm->arg_end;
    up_read(&mm->mmap_lock);

    len = arg_end - arg_start;
    if (len > MAX_CMDLINE_LENGTH - 1)
        len = MAX_CMDLINE_LENGTH - 1;

    if (len <= 0 || access_process_vm(task, arg_start, cmdline, len, 0) != len) {
        mmput(mm);
        kfree(cmdline);
        return NULL;
    }

    cmdline[len] = '\0';
    for (i = 0, p = cmdline; i < len; i++)
        if (p[i] == '\0') p[i] = ' ';

    mmput(mm);
    return cmdline;
}

/* /proc reader */
static int sysinfo_show(struct seq_file *m, void *v)
{
    struct sysinfo si;
    struct task_struct *task;
    unsigned long j_total = jiffies;
    unsigned long totalram_kb, freeram_kb;
    int first = 1;

    si_meminfo(&si);

    /* páginas -> KB */
    totalram_kb = si.totalram * (PAGE_SIZE / 1024);
    freeram_kb  = si.freeram  * (PAGE_SIZE / 1024);

    seq_printf(m, "{\n");
    seq_printf(m, "  \"Totalram\": %lu,\n", totalram_kb);
    seq_printf(m, "  \"Freeram\": %lu,\n",  freeram_kb);
    seq_printf(m, "  \"Procs\": %d,\n",     si.procs);
    seq_printf(m, "  \"Processes\": [\n");

    for_each_process(task) {
        unsigned long vsz_kb = 0;
        unsigned long rss_kb = 0;
        unsigned long mem_permille = 0; /* 0..1000 => 1 decimal */
        unsigned long cpu_centi = 0;    /* 0..10000 => 2 decimales */
        char *cmdline = NULL;

        if (task->mm) {
            vsz_kb = task->mm->total_vm << (PAGE_SHIFT - 10);
            rss_kb = get_mm_rss(task->mm) << (PAGE_SHIFT - 10);
            if (totalram_kb > 0)
                mem_permille = (rss_kb * 1000UL) / totalram_kb;
        }

        if (j_total > 0) {
            unsigned long t = task->utime + task->stime;
            cpu_centi = (t * 10000UL) / j_total; /* aproximado */
        }

        cmdline = get_process_cmdline(task);

        if (!first) seq_printf(m, ",\n"); else first = 0;

        seq_printf(m, "    {\n");
        seq_printf(m, "      \"PID\": %d,\n", task->pid);
        seq_printf(m, "      \"Name\": \"%s\",\n", task->comm);
        seq_printf(m, "      \"State\": \"%s\",\n", task_state_short(task));
        seq_printf(m, "      \"Cmdline\": \"%s\",\n", cmdline ? cmdline : "N/A");
        seq_printf(m, "      \"vsz\": %lu,\n", vsz_kb);
        seq_printf(m, "      \"rss\": %lu,\n", rss_kb);
        seq_printf(m, "      \"Memory_Usage\": %lu.%lu,\n", mem_permille / 10, mem_permille % 10);
        seq_printf(m, "      \"CPU_Usage\": %lu.%02lu\n",   cpu_centi / 100,   cpu_centi % 100);
        seq_printf(m, "    }");

        if (cmdline) kfree(cmdline);
    }

    seq_printf(m, "\n  ]\n}\n");
    return 0;
}

/* open /proc */
static int sysinfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, sysinfo_show, NULL);
}

/* operaciones /proc */
static const struct proc_ops sysinfo_ops = {
    .proc_open    = sysinfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

/* init/exit */
static int __init sysinfo_init(void)
{
    if (!proc_create(PROC_NAME, 0444, NULL, &sysinfo_ops)) {
        pr_err("sysinfo_json: no se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    pr_info("sysinfo_json: modulo cargado (/proc/%s)\n", PROC_NAME);
    return 0;
}

static void __exit sysinfo_exit(void)
{
    remove_proc_entry(PROC_NAME, NULL);
    pr_info("sysinfo_json: modulo desinstalado (/proc/%s)\n", PROC_NAME);
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);

```

#### continfo_so1_202203009
Dicho modulo se encarga de de mostrar todos los procesos del sistema, asi como su informacion en formato json, esa muestra el PID, nombre, linea de comando que se ejecuto, vsz, rss, porcentaje de memoria usada y porcentaje de cpu.
```bash
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/string.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/jiffies.h>
#include <linux/uaccess.h>
#include <linux/sched/signal.h>
#include <linux/slab.h>
#include <linux/sched/mm.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Mateo Estuardo Diego Noriega");
MODULE_DESCRIPTION("Modulo que expone informacion de procesos de contenedores en JSON");
MODULE_VERSION("1.0");

#define PROC_NAME "continfo_so1_202203009"
#define MAX_CMDLINE_LENGTH 256
#define CONTAINER_ID_MAX   64  /* mostramos hasta 64 chars */

/* ---- Cmdline del proceso (caller debe kfree) ---- */
static char *get_process_cmdline(struct task_struct *task)
{
    struct mm_struct *mm;
    char *cmdline, *p;
    unsigned long arg_start, arg_end;
    int i, len;

    cmdline = kmalloc(MAX_CMDLINE_LENGTH, GFP_KERNEL);
    if (!cmdline)
        return NULL;

    mm = get_task_mm(task);
    if (!mm) {
        kfree(cmdline);
        return NULL;
    }

    down_read(&mm->mmap_lock);
    arg_start = mm->arg_start;
    arg_end   = mm->arg_end;
    up_read(&mm->mmap_lock);

    len = arg_end - arg_start;
    if (len > MAX_CMDLINE_LENGTH - 1)
        len = MAX_CMDLINE_LENGTH - 1;

    if (len <= 0 || access_process_vm(task, arg_start, cmdline, len, 0) != len) {
        mmput(mm);
        kfree(cmdline);
        return NULL;
    }

    cmdline[len] = '\0';
    for (i = 0, p = cmdline; i < len; i++)
        if (p[i] == '\0') p[i] = ' ';

    mmput(mm);
    return cmdline;
}

/* ---- Heurística: ¿es shim de contenedor? ---- */
static bool is_container_shim(const char *comm)
{
    /* Docker/Containerd modernos usan "containerd-shim-runc-v2".
       Algunos setups viejos: "containerd-shim". */
    return (strcmp(comm, "containerd-shim-runc") == 0) ||
           (strcmp(comm, "containerd-shim-runc-v2") == 0) ||
           (strcmp(comm, "containerd-shim") == 0);
}

/* ---- Extrae ContainerID desde la cmdline
   Preferimos token después de "--id". Si no, tomamos la primera
   "palabra" predominantemente hex de longitud >= 12 (Docker IDs). ---- */
static void extract_container_id(const char *cmdline, char *out, size_t outlen)
{
    const char *p, *end;
    size_t n;

    if (!cmdline || !*cmdline) {
        strscpy(out, "N/A", outlen);
        return;
    }

    n = strlen(cmdline);
    p = cmdline;
    end = cmdline + n;

    /* 1) Busca "--id" */
    while (p < end) {
        while (p < end && *p == ' ') p++;
        if (p + 4 <= end && strncmp(p, "--id", 4) == 0) {
            p += 4;
            while (p < end && *p == ' ') p++;
            /* copiar token siguiente */
            {
                const char *start = p;
                while (p < end && *p != ' ') p++;
                if (p > start) {
                    size_t len = p - start;
                    if (len >= outlen) len = outlen - 1;
                    memcpy(out, start, len);
                    out[len] = '\0';
                    return;
                }
            }
        }
        while (p < end && *p != ' ') p++;
    }

    /* 2) Si no hay --id, busca primera palabra mayormente hex (>=12) */
    p = cmdline;
    while (p < end) {
        while (p < end && *p == ' ') p++;
        const char *start = p;
        int hex = 0, total = 0;

        while (p < end && *p != ' ') {
            char c = *p;
            if ((c >= '0' && c <= '9') ||
                (c >= 'a' && c <= 'f') ||
                (c >= 'A' && c <= 'F')) {
                hex++;
            }
            total++;
            p++;
        }
        if (total >= 12 && hex * 10 >= total * 8) { /* >=80% hex */
            size_t len = p - start;
            if (len >= outlen) len = outlen - 1;
            memcpy(out, start, len);
            out[len] = '\0';
            return;
        }
        while (p < end && *p == ' ') p++;
    }

    strscpy(out, "N/A", outlen);
}

/* ---- /proc reader ---- */
static int continfo_show(struct seq_file *m, void *v)
{
    struct sysinfo si;
    struct task_struct *task;

    unsigned long j_total = jiffies;
    unsigned long totalram_kb, freeram_kb;
    int first = 1;

    si_meminfo(&si);
    totalram_kb = si.totalram * (PAGE_SIZE / 1024);
    freeram_kb  = si.freeram  * (PAGE_SIZE / 1024);

    seq_printf(m, "{\n");
    seq_printf(m, "  \"Totalram\": %lu,\n", totalram_kb);
    seq_printf(m, "  \"Freeram\": %lu,\n",  freeram_kb);
    seq_printf(m, "  \"Processes\": [\n");

    /* Recorre todos; filtra shims */
    for_each_process(task) {
        struct task_struct *target = NULL; /* proceso medido (hijo si hay) */
        unsigned long vsz_kb = 0, rss_kb = 0;
        unsigned long mem_permille = 0; /* 0..1000 => 1 decimal */
        unsigned long cpu_centi = 0;    /* 0..10000 => 2 decimales */
        char *cmdline_shim = NULL, *cmdline_target = NULL;
        char container_id[CONTAINER_ID_MAX];

        if (!is_container_shim(task->comm))
            continue;

        /* cmdline del shim y su ID */
        cmdline_shim = get_process_cmdline(task);
        extract_container_id(cmdline_shim ? cmdline_shim : "", container_id, sizeof(container_id));

        /* intenta usar el primer hijo con mm (proceso del contenedor) */
        {
            struct task_struct *child;
            list_for_each_entry(child, &task->children, sibling) {
                if (child->mm) { target = child; break; }
            }
        }
        if (!target) target = task; /* fallback: medir el shim */

        /* Métricas memoria */
        if (target->mm) {
            vsz_kb = target->mm->total_vm << (PAGE_SHIFT - 10);
            rss_kb = get_mm_rss(target->mm) << (PAGE_SHIFT - 10);
            if (totalram_kb > 0)
                mem_permille = (rss_kb * 1000UL) / totalram_kb;
        }

        /* Métrica CPU (aprox) */
        if (j_total > 0) {
            unsigned long t = target->utime + target->stime;
            cpu_centi = (t * 10000UL) / j_total; /* 2 decimales */
        }

        cmdline_target = get_process_cmdline(target);

        if (!first) seq_printf(m, ",\n"); else first = 0;

        seq_printf(m, "    {\n");
        seq_printf(m, "      \"ShimPID\": %d,\n", task->pid);
        seq_printf(m, "      \"ShimName\": \"%s\",\n", task->comm);
        seq_printf(m, "      \"ContainerID\": \"%s\",\n", container_id);
        seq_printf(m, "      \"PID\": %d,\n", target->pid);
        seq_printf(m, "      \"Name\": \"%s\",\n", target->comm);
        seq_printf(m, "      \"Cmdline\": \"%s\",\n", cmdline_target ? cmdline_target : (cmdline_shim ? cmdline_shim : "N/A"));
        seq_printf(m, "      \"vsz\": %lu,\n", vsz_kb);
        seq_printf(m, "      \"rss\": %lu,\n", rss_kb);
        seq_printf(m, "      \"Memory_Usage\": %lu.%lu,\n", mem_permille / 10, mem_permille % 10);
        seq_printf(m, "      \"CPU_Usage\": %lu.%02lu\n",   cpu_centi / 100,   cpu_centi % 100);
        seq_printf(m, "    }");

        if (cmdline_target) kfree(cmdline_target);
        if (cmdline_shim)   kfree(cmdline_shim);
    }

    seq_printf(m, "\n  ]\n}\n");
    return 0;
}

/* open /proc */
static int continfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, continfo_show, NULL);
}

static const struct proc_ops continfo_ops = {
    .proc_open    = continfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

/* init/exit */
static int __init continfo_init(void)
{
    if (!proc_create(PROC_NAME, 0444, NULL, &continfo_ops)) {
        pr_err("continfo_json: no se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    pr_info("continfo_json: modulo cargado (/proc/%s)\n", PROC_NAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    remove_proc_entry(PROC_NAME, NULL);
    pr_info("continfo_json: modulo desinstalado (/proc/%s)\n", PROC_NAME);
}

module_init(continfo_init);
module_exit(continfo_exit);
```

---

### Compilacion de modulo
Una vez creados los modulos de kernel, para poder compilarlos debe de crearse un archivo Makefile con el siguiente contenido.

```bash
obj-m += sysinfo.o

all:
   make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules

clean:
   make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean
```

Este Makefile se encargara de compilar ambos modulos de kernel, sin embargo es importante mecnionar que para evitar problemas se recomiendo utilizar la version 12 de gcc, asi como tambien tomar en cuenta la ruta en la que se enuentra el makefile, puesto que si para acceder a esa ruta existe alguna carpeta con espacios entre el nombre, debera de hacerse una pequenia modificacion en el makefile para que reconozca correctamente la ruta, caso contrario, esta dara error.

Una vez se haya creado correctamente el Makefile, la manera de correrlo es abriendo una terminal en la ubicacion del makefile y ejecutar los siguientes comandos:

```bash
make
```

Si todo es correcto se generará un archivo con la extensión .ko. Una vez hecho esto, se debera cargar el modulo en el kernel, para ello se hara uso de los siguientes comandos.

```bash
sudo insmod sysinfo_so1_202203009.ko
sudo insmod continfo_so1_202203009.ko
```

Ahora, para verificar que el modulo se haya cargado se usa:

```bash
lsmod | grep sysinfo_so1_202203009
lsmod | grep continfo_so1_202203009
```

Una vez se haya terminado de utilizar el modulo, este debe removerse, para ello se usa el siguiente comando:

```bash
sudo rmmod sysinfo_so1_202203009
sudo rmmod continfo_so1_202203009
```

Para verificar si el modulo fue removido correctamente se utiliza el siguiente comando.

```bash
lsmod | grep sysinfo
```

---

## Daemon
Una vez realizado lo anterior comenzaremos con el desarrollo del daemon, para ellos, se creo una carpeta llamada go-daemon, en la cual, se creo el main, el cual se compila con el siguiente comando:

```bash
go build -o /usr/local/bin/daemon_202203009 main.go
```

Posterior a ello, ejecutamos el siguiente comando:

```bash
sudo nano /etc/systemd/system/daemon_202203009.service
```

Dicho comando nos abre un editor de texto, en el cual se pega el siguiente contenido:

```bash
[Unit]
Description=Mi daemon en Go
After=network.target

[Service]
ExecStart=/usr/local/bin/daemon_202203009
Restart=always

[Install]
WantedBy=multi-user.target
```

Posterior a ello se activa y arranca el daemon con el siguiente comando:

```bash
sudo systemctl daemon-reload        # recargar systemd
sudo systemctl enable --now daemon_202203009
```

Una vez realizados los pasos anteriores podesmos verificar que ya esta corriendo con ayuda de los siguientes comandos:

Estado del servicio:

```bash
systemctl status daemon_202203009
```

Con el siguiente comando puede observarse si el daemon esta incializado o no:

```bash
journalctl -u daemon_202203009 -f
```

Logs en el archivo que configuramos (/var/log/daemon_202203009.log):

```bash
tail -f /var/log/daemon_202203009.log
```
Por ultimo, para apagar el daemon usamos los siguientes comandos:

```bash
sudo systemctl stop daemon_202203009
sudo systemctl disable daemon_202203009
```

Cada vez que un cambio fue realizado en el `main.go` se hizo uso de los siguientes comandos:

```bash
go build -o daemon_202203009 main.go
sudo install -m 0755 ./daemon_202203009 /usr/local/bin/
sudo systemctl restart daemon_202203009
sudo tail -f /var/log/daemon_202203009.log   # o journalctl -u ...
```
---

## CRONJOB
Para el apartado del cronjob se creo el directorio llamado `bash` el cual contiene los siguientes archivos:

```
└── 📁bash
    ├── cron_spawn_containers.sh
    ├── install_cron.sh
    ├── load_modules.sh
    └── remove_cron.sh
```

### install_cron.sh
Este archivo es el encargado de instalar el cronjob en el sistema, que sera el encargado de ejecutar el spawn de contenedores por minuto.

El Spawn se encarga de apuntar al script que crea contenedores
```bash
SPAWN="${SCRIPT_DIR}/cron_spawn_containers.sh"
```

Con estas otras lineas garantizamos que los permisos de ejecucion del spawn y que el log existan y sean legibles.

```bash
chmod +x "$SPAWN"
touch "$LOGFILE"
chmod 0644 "$LOGFILE"
```

### cron_spawn_containers.sh
Este archivo es el encargado de crear contenedores cada minuto.

El Spawn se encarga de apuntar al script que crea contenedores. Asimismo, la siguiente parte del codigo permite como "tunear" sin tener que editar el script usando variables de entorno

Para la creacion de los 10 contenedores se utilizo el siguiente codigo, el cual se encarga de crear 10 contenedores de alto/bajo y consumo mixto de forma aleatoria

- High-CPU (alto consumo de CPU)

- High-RAM (alto consumo de RAM)

- Low (bajo consumo de RAM y CPU)

Además, etiqueta todos los contenedores para poder filtrarlos y gestionarlos fácilmente desde el daemon (proyecto2=1, tier=high/low, mode=cpu/ram/low).

```bash
# Total exacto por minuto (default 10)
TOTAL="${P2_TOTAL_SPAWN:-10}"

# High vs Low (aleatorio si no viene por env)
BATCH_HI="${P2_BATCH_HI:-$((RANDOM % (TOTAL+1)))}"
BATCH_LO="$((TOTAL - BATCH_HI))"

# High-CPU vs High-RAM (aleatorio si no viene por env)
if [[ -n "${P2_BATCH_HI_CPU:-}" && -n "${P2_BATCH_HI_RAM:-}" ]]; then
  BATCH_HI_CPU="$P2_BATCH_HI_CPU"
  BATCH_HI_RAM="$P2_BATCH_HI_RAM"
else
  BATCH_HI_CPU="$((RANDOM % (BATCH_HI+1)))"
  BATCH_HI_RAM="$((BATCH_HI - BATCH_HI_CPU))"
fi

IMAGE_HI_CPU="${P2_IMAGE_HI_CPU:-p2/high-cpu:1}"
IMAGE_HI_RAM="${P2_IMAGE_HI_RAM:-p2/high-ram:1}"
IMAGE_LO="${P2_IMAGE_LO:-p2/low:1}"
```
Con ese estracto de codigo se pueden generar los contenedores de manera random, sin sobrepasar la cantidad de contenedores especificada en el enunciado. Esto se logro por medio del spawn de contenedores, donde se crearon variables que le asginan valores random a las imagenes de docker que se utilizaran.


Adicionalmente, para poder visualizar los ultimos logs nos ayudamos del siguiente comando:

```bash
sudo tail -n 100 /var/log/cron_spawn.log
```
## Imagenes

```
└── 📁imagenes
    └── 📁high-cpu
        ├── Dockerfile
    └── 📁high-ram
        ├── Dockerfile
    └── 📁low
        └── Dockerfile
```

Para la creacion de las imagenes se utilizo los siguientes dockerfiles.

### high-cpu
```bash
FROM alpine:3.20
RUN apk add --no-cache coreutils
ENV CPU_WORKERS=1
# Lanza N procesos "yes" para forzar CPU; espera a todos
CMD ["sh","-c","i=0; while [ $i -lt ${CPU_WORKERS} ]; do yes > /dev/null & i=$((i+1)); done; wait"]
```

### high-ram
```bash
FROM alpine:3.20
RUN apk add --no-cache stress-ng
ENV RAM_MB=512 VM_WORKERS=1
# --vm-keep mantiene la memoria asignada; sin timeout
CMD ["sh","-c","exec stress-ng --vm ${VM_WORKERS} --vm-bytes ${RAM_MB}M --vm-keep --timeout 0"]
```

### low
```bash
FROM alpine:3.20
ENV SLEEP_SECS=1800
CMD ["sh","-c","exec sleep ${SLEEP_SECS}"]
```

Las tres imagenes reciben datos desde el spwan y de esta manera los contenedores se basan en imagenes con consumos variables cada una.

### remove_cron.sh
Este archivo se encarga de eliminar el cron cada vez que finalice la ejeucion del programa.

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPAWN="${SCRIPT_DIR}/cron_spawn_containers.sh"

# Si no hay crontab previo, no falla
if crontab -l >/dev/null 2>&1; then
  crontab -l | grep -v "$SPAWN" | crontab -
fi

echo "[remove_cron] Eliminada entrada del cron para ${SPAWN}"
```

Este es llamado desde el main mediante la funcion `defer`, la cual le indica al programa que se era el metodo ejecutado antes de cerrar la ejecucion.

### load_modules.sh
Este archivo se encarga de cargar los modulos de sysinfo y continfo cada vez que se ejecuta el programa. Siendo este el primer archivo ejecutado al correr el main del proyecto.

```js
    // 1) Cargar módulos del kernel vía script
	if err := runScript(loadModulesScript, 60*time.Second); err != nil {
		logger.Printf("load_modules.sh falló: %v", err)
	}
```

---
## Grafana
Para la parte de visualizacion de datos se utilizo grafana su uso un archivo llamado docker-compose.yml, con el siguiente contenido:

```bash
services:
  grafana:
    image: grafana/grafana:10.4.5
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_INSTALL_PLUGINS=frser-sqlite-datasource
    volumes:
      - grafana_data:/var/lib/grafana
      - /var/lib/proyecto2:/var/lib/proyecto2
      - ./provisioning/datasources:/etc/grafana/provisioning/datasources:ro
      - ./provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
    restart: unless-stopped

volumes:
  grafana_data:
```

Con dicho archivo se tiene accesos a grafana. Para el ingreso debe de conectarse al puerto 3000 en localhost, e ingresar las credenciales `admin` y `admin123`. Una vez realizado esto se crean los dashboards.

Se creo un dashboard para la visualizacion de los datos del host y otro dashboard para la visualizacion de los contenedores.

## go-daemon
### Estructura
```
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
    └── main.go
```
dentro del directorio encontraremos la logica del sistema, el `main.go`.

Para tomar las decisiones sobre que contenedores se matan y cuales no se creo un archivo llamado decider, el cual se encarga de analizar los datos, y en base a ellos mata contenedores:

```bash
package decider

import (
	"sort"
	"strconv"
	"strings"

	"proyecto2/daemon/internal/proc"
)

// candidato consolidado por contenedor
type cand struct {
	ID       string
	Name     string
	CPUPct   float64
	MemPct   float64
	RSS      uint64
	VSZ      uint64
	ShimPID  int
	ShimName string
}

type Decision struct {
	KeepIDs []string
	KillIDs []string
	Reason  map[string]string // reason[cid] = explicación corta
}

// PickKeepSet selecciona 2 "altos" + 3 "bajos" y devuelve qué matar (respetando protegidos).
// Reglas:
// - Protegidos nunca van en KillIDs (aunque no estén en Keep).
// - Si hay <5 contenedores, no matamos (Keep = todo, Kill = vacío).
// - "Altos": mayor CPU, luego Mem, luego RSS, luego VSZ.
// - "Bajos": menor CPU, luego Mem, luego RSS, luego VSZ.
func PickKeepSet(snap proc.ContSnapshot, protectIDs []string, protectNames []string) Decision {
	// normaliza listas de protección
	protID := make(map[string]struct{})
	for _, id := range protectIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			protID[id] = struct{}{}
		}
	}
	var protNames []string
	for _, n := range protectNames {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" {
			protNames = append(protNames, n)
		}
	}

	// indexa por ContainerID (si viene vacío, cae a nombre/pid como fallback)
	byID := make(map[string]cand)
	for _, p := range snap.Processes {
		id := canonicalID(p)
		c := cand{
			ID:       id,
			Name:     p.Name,
			CPUPct:   p.CPUPct,
			MemPct:   p.MemPct,
			RSS:      p.RSS,
			VSZ:      p.VSZ,
			ShimPID:  p.ShimPID,
			ShimName: p.ShimName,
		}
		if prev, ok := byID[id]; ok {
			if better(c, prev) {
				byID[id] = c
			}
		} else {
			byID[id] = c
		}
	}

	// arma el slice de candidatos
	cands := make([]cand, 0, len(byID))
	for _, c := range byID {
		cands = append(cands, c)
	}

	total := len(cands)
	dec := Decision{Reason: make(map[string]string)}

	// si no hay suficientes, mantenemos todo
	if total <= 5 {
		for _, c := range cands {
			dec.KeepIDs = append(dec.KeepIDs, c.ID)
			dec.Reason[c.ID] = "insufficient_containers"
		}
		// KillIDs vacío
		return dec
	}

	// ordenamientos
	highs := make([]cand, len(cands))
	copy(highs, cands)
	sort.SliceStable(highs, func(i, j int) bool {
		// descendente: CPU, Mem, RSS, VSZ
		if highs[i].CPUPct != highs[j].CPUPct {
			return highs[i].CPUPct > highs[j].CPUPct
		}
		if highs[i].MemPct != highs[j].MemPct {
			return highs[i].MemPct > highs[j].MemPct
		}
		if highs[i].RSS != highs[j].RSS {
			return highs[i].RSS > highs[j].RSS
		}
		return highs[i].VSZ > highs[j].VSZ
	})

	lows := make([]cand, len(cands))
	copy(lows, cands)
	sort.SliceStable(lows, func(i, j int) bool {
		// ascendente: CPU, Mem, RSS, VSZ
		if lows[i].CPUPct != lows[j].CPUPct {
			return lows[i].CPUPct < lows[j].CPUPct
		}
		if lows[i].MemPct != lows[j].MemPct {
			return lows[i].MemPct < lows[j].MemPct
		}
		if lows[i].RSS != lows[j].RSS {
			return lows[i].RSS < lows[j].RSS
		}
		return lows[i].VSZ < lows[j].VSZ
	})

	keepSet := make(map[string]struct{})

	// helper: protegido por ID o por substring del nombre
	isProtected := func(c cand) bool {
		if _, ok := protID[c.ID]; ok {
			return true
		}
		name := strings.ToLower(c.Name)
		for _, sub := range protNames {
			if sub != "" && strings.Contains(name, sub) {
				return true
			}
		}
		return false
	}

	// elige 2 altos
	for _, c := range highs {
		if len(keepSet) >= 2 {
			break
		}
		keepSet[c.ID] = struct{}{}
		dec.Reason[c.ID] = "high_rank"
	}

	// elige 3 bajos (sin duplicar)
	for _, c := range lows {
		if len(keepSet) >= 5 {
			break
		}
		if _, ok := keepSet[c.ID]; ok {
			continue
		}
		keepSet[c.ID] = struct{}{}
		if _, ok := dec.Reason[c.ID]; !ok {
			dec.Reason[c.ID] = "low_rank"
		}
	}

	// pasa keepSet a slice
	for id := range keepSet {
		dec.KeepIDs = append(dec.KeepIDs, id)
	}

	// calcula KillIDs = todos los demás (pero NUNCA protegidos)
	for _, c := range cands {
		if _, ok := keepSet[c.ID]; ok {
			continue
		}
		if isProtected(c) {
			dec.Reason[c.ID] = "protected"
			continue
		}
		dec.KillIDs = append(dec.KillIDs, c.ID)
		dec.Reason[c.ID] = "not_in_keep"
	}

	// ordena para tener salida determinista
	sort.Strings(dec.KeepIDs)
	sort.Strings(dec.KillIDs)

	return dec
}

func better(a, b cand) bool {
	// criterio "mejor" para representar al contenedor
	if a.CPUPct != b.CPUPct {
		return a.CPUPct > b.CPUPct
	}
	if a.MemPct != b.MemPct {
		return a.MemPct > b.MemPct
	}
	if a.RSS != b.RSS {
		return a.RSS > b.RSS
	}
	return a.VSZ > b.VSZ
}

func canonicalID(p proc.ContProc) string {
	// Preferimos ContainerID; si viene vacío, caemos a shim o nombre con PID
	if p.ContainerID != "" {
		return p.ContainerID
	}
	if p.ShimPID != 0 {
		return "shim:" + itoa(p.ShimPID)
	}
	if p.PID != 0 {
		return "pid:" + itoa(p.PID)
	}
	if p.Name != "" {
		return "name:" + strings.ToLower(p.Name)
	}
	return "unknown"
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
```
Asimismo, todos los datos mostrados en grafana son obtenidos de la base de datos:

```go
package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct{ *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// --- Ajustes recomendados para concurrencia y rendimiento ---
	// WAL permite lecturas concurrentes mientras se escribe.
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	// Menos fsyncs, suficiente para logs/métricas.
	_, _ = db.Exec(`PRAGMA synchronous=NORMAL;`)
	// Esperar hasta 5s si la DB está ocupada (evita "database is locked").
	_, _ = db.Exec(`PRAGMA busy_timeout=5000;`)

	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS system_metrics (
  ts INTEGER NOT NULL,
  pid INTEGER, name TEXT, state TEXT, cmdline TEXT,
  vsz_kb INTEGER, rss_kb INTEGER, mem_pct REAL, cpu_pct REAL
);
CREATE TABLE IF NOT EXISTS container_metrics (
  ts INTEGER NOT NULL,
  container_id TEXT, shim_pid INTEGER, shim_name TEXT,
  pid INTEGER, name TEXT, cmdline TEXT,
  vsz_kb INTEGER, rss_kb INTEGER, mem_pct REAL, cpu_pct REAL
);
CREATE TABLE IF NOT EXISTS actions_log (
  ts INTEGER NOT NULL,
  action TEXT, container_id TEXT, reason TEXT, details TEXT
);
CREATE TABLE IF NOT EXISTS host_metrics (
  ts INTEGER NOT NULL,
  totalram_kb INTEGER,
  freeram_kb INTEGER
);
CREATE INDEX IF NOT EXISTS idx_host_ts ON host_metrics(ts);
-- Índices útiles
CREATE INDEX IF NOT EXISTS idx_sys_ts ON system_metrics(ts);
CREATE INDEX IF NOT EXISTS idx_cont_ts ON container_metrics(ts);
CREATE INDEX IF NOT EXISTS idx_cont_cid ON container_metrics(container_id);
`)
	return err
}

func nowTS() int64 { return time.Now().Unix() }

// ===== Insert helpers "auto-ts" (las tuyas originales) =====

func (s *Store) InsertSystemProc(pid int, name, state, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO system_metrics (ts,pid,name,state,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		nowTS(), pid, name, state, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) InsertContainerProc(cid string, shimPID int, shimName string, pid int, name, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO container_metrics (ts,container_id,shim_pid,shim_name,pid,name,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		nowTS(), cid, shimPID, shimName, pid, name, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) LogAction(action, cid, reason, details string) error {
	_, err := s.Exec(
		`INSERT INTO actions_log (ts,action,container_id,reason,details) VALUES (?,?,?,?,?)`,
		nowTS(), action, cid, reason, details,
	)
	return err
}

// ===== Opcionales: variantes con ts explícito (por si las quieres usar) =====

func (s *Store) InsertSystemProcWithTS(ts int64, pid int, name, state, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO system_metrics (ts,pid,name,state,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		ts, pid, name, state, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) InsertContainerProcWithTS(ts int64, cid string, shimPID int, shimName string, pid int, name, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO container_metrics (ts,container_id,shim_pid,shim_name,pid,name,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		ts, cid, shimPID, shimName, pid, name, cmd, vsz, rss, mem, cpu,
	)
	return err
}

```

## Levantar servicios

Para levantar los servicios basta con levantar grafana y el daemon, para ello nos apoyaremos de los siguientes comandos:

```bash
#En el directorio de dashboard
docker compose -p p2dash up -d

#Los siguientes comandos son para verificar que esta arriba
docker compose -p p2dash ps
docker compose -p p2dash logs -n 50

#Para levantar el daemon debemos situarnos en go-daemon
sudo systemctl daemon-reload        # recargar systemd
sudo systemctl enable --now daemon_202203009

# Para verificar su estado utilizamos el siguiente comando
systemctl status daemon_202203009
journalctl -u daemon_202203009 -f # para ver si esta inicializado
tail -f /var/log/daemon_202203009.log # para ver logs
```
Una vez hayamos terminado de utilizar el programa podemos bajar los servicios con los siguientes comandos:

```bash
# Para parar grafana
docker compose -p p2dash down

# Para parar el daemon y deshabilitarlo
sudo systemctl stop daemon_202203009
sudo systemctl disable daemon_202203009
```
Recordar para que funcione primero debe de levantarse grafana y despues el daemon.