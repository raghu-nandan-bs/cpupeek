
// +build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#ifndef IFNAMSIZ
#define IFNAMSIZ 16
#endif

struct runtime_info_t{
    char comm[16];
    __u64 cpu;
    __u64 runtime;
};

// this will be updated in go program
// Accept user provided PID, cpu to track
const volatile int64_t pid, cpu = -1;

struct {
    __uint(type, BPF_MAP_TYPE_LRU_PERCPU_HASH);
    __uint(max_entries, 200); // we cant really display anything more on screen meaningfully. 
    __type(key, __u32); // __pid
    __type(value, struct runtime_info_t); 
}runtime_arr SEC(".maps");

SEC("tp_btf/sched_stat_runtime")

int BPF_PROG(
    trace_sched_stat_runtime,
    __u64 vruntime,
    __u64 runtime)
{   

    struct runtime_info_t __info;
    bpf_get_current_comm(&__info.comm, sizeof(__info.comm));
   
    __info.cpu = bpf_get_smp_processor_id();
    __u32 __pid = bpf_get_current_pid_tgid();

    /* If user has provided PID or CPU to track specifically,
    we ignore the others.
    */
    if( pid != -1 && __pid != pid) {
        return 0;
    }
    if (cpu != -1 && __info.cpu != cpu) {
        return 0;
    }
    struct runtime_info_t *__info_returned = bpf_map_lookup_elem(&runtime_arr, &__pid);
    if (!__info_returned) {
        __info.runtime = runtime;
    } else {
        __info.runtime = __info_returned->runtime + runtime;
    }

    int err = bpf_map_update_elem(&runtime_arr, &__pid, &__info, BPF_ANY);
    if (err < 0) {
        bpf_printk("[Err] update runtime_arr failed: %d\n",err);
    }
    return 0;
}

char LICENSE[] SEC("license") = "GPL";