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

/* Escapa una cadena para JSON y la escribe en seq_file */
static void seq_puts_json_escaped(struct seq_file *m, const char *s)
{
    const unsigned char *p = (const unsigned char *)s;
    if (!p) return;
    for (; *p; p++) {
        unsigned char c = *p;
        switch (c) {
        case '\"': seq_puts(m, "\\\""); break;
        case '\\': seq_puts(m, "\\\\"); break;
        case '\n': seq_puts(m, "\\n");  break;
        case '\r': seq_puts(m, "\\r");  break;
        case '\t': seq_puts(m, "\\t");  break;
        default:
            if (c < 0x20) {
                seq_printf(m, "\\u%04x", c);
            } else {
                seq_putc(m, c);
            }
        }
    }
}


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
    int proc_count = 0; /* <--- NUEVO */

    si_meminfo(&si);

    /* cuenta procesos (pase rápido sólo para el total) */
    for_each_process(task) {
        proc_count++;
    }

    /* páginas -> KB */
    totalram_kb = si.totalram * (PAGE_SIZE / 1024);
    freeram_kb  = si.freeram  * (PAGE_SIZE / 1024);

    seq_printf(m, "{\n");
    seq_printf(m, "  \"Totalram\": %lu,\n", totalram_kb);
    seq_printf(m, "  \"Freeram\": %lu,\n",  freeram_kb);
    seq_printf(m, "  \"Procs\": %d,\n",     proc_count); /* <--- NUEVO */
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
        seq_puts(m,  "      \"Name\": \"");
        seq_puts_json_escaped(m, task->comm);
        seq_puts(m,  "\",\n");
        seq_printf(m, "      \"State\": \"%s\",\n", task_state_short(task));
        seq_puts(m,  "      \"Cmdline\": \"");
        seq_puts_json_escaped(m, cmdline ? cmdline : "N/A");
        seq_puts(m,  "\",\n");
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
