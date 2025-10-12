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
