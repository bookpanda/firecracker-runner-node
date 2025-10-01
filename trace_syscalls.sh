#!/bin/bash

if [ -n "$1" ]; then
    PID="$1"
    echo "Tracing PID: $PID"
else
    echo "PID not found!"
    exit 1
fi

sudo bpftrace -e '
BEGIN { 
    printf("Tracing network syscalls for PID '$PID'...\n"); 
}

tracepoint:syscalls:sys_enter_* /pid == '$PID'/ {
    @interval[comm, probe]++;
    @total[comm, probe]++;
}

interval:s:2 {
    printf("\n--- syscall counts (last 2s) ---\n");
    print(@interval);
    clear(@interval);
    printf("\n--- cumulative syscall counts ---\n");
    print(@total);
}

END { 
    printf("\n=== FINAL cumulative syscall counts ===\n"); 
    print(@total);
    printf("=== END OF TRACE ===\n");
}
' 
