#include <linux/module.h>
#define INCLUDE_VERMAGIC
#include <linux/build-salt.h>
#include <linux/elfnote-lto.h>
#include <linux/export-internal.h>
#include <linux/vermagic.h>
#include <linux/compiler.h>

#ifdef CONFIG_UNWINDER_ORC
#include <asm/orc_header.h>
ORC_HEADER;
#endif

BUILD_SALT;
BUILD_LTO_INFO;

MODULE_INFO(vermagic, VERMAGIC_STRING);
MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};

#ifdef CONFIG_RETPOLINE
MODULE_INFO(retpoline, "Y");
#endif



static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0x46b0be46, "single_open" },
	{ 0x4c03a563, "random_kmalloc_seed" },
	{ 0x1bff00c8, "kmalloc_caches" },
	{ 0xd0c3484c, "kmalloc_trace" },
	{ 0xda2a5bc3, "get_task_mm" },
	{ 0x668b19a1, "down_read" },
	{ 0x53b954a2, "up_read" },
	{ 0xf279d66e, "access_process_vm" },
	{ 0xdcefc2e4, "mmput" },
	{ 0x37a0cba, "kfree" },
	{ 0x63cd0125, "remove_proc_entry" },
	{ 0x15ba50a6, "jiffies" },
	{ 0x40c7247c, "si_meminfo" },
	{ 0xc00e2b80, "seq_printf" },
	{ 0xa54c0f5b, "init_task" },
	{ 0xe2d5255a, "strcmp" },
	{ 0x754d539c, "strlen" },
	{ 0x5a921311, "strncmp" },
	{ 0xf0fdf6cb, "__stack_chk_fail" },
	{ 0xc1d9b323, "seq_read" },
	{ 0x7369f212, "seq_lseek" },
	{ 0x8e66928c, "single_release" },
	{ 0xbdfb6dbb, "__fentry__" },
	{ 0x2dbde678, "proc_create" },
	{ 0x122c3a7e, "_printk" },
	{ 0x5b8239ca, "__x86_return_thunk" },
	{ 0xe2fd41e5, "module_layout" },
};

MODULE_INFO(depends, "");


MODULE_INFO(srcversion, "135DAFD40AD3688B75A9E05");
