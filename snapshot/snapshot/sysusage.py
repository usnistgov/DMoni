import psutil
import simplejson
from rfc3339 import rfc3339
from datetime import datetime

def usages(attrs):

    """usage collects system's resource usages, including cpu
    virtual memory, swap memory, disk IO and network IO.
    """

    info = {}

    if "cpu" in attrs: 
        info["cpu_times"] = psutil.cpu_times()
        info["cpu_count"] = psutil.cpu_count()
        # Measure cpu usage percent in a small time interval
        info["cpu_percent"] = psutil.cpu_percent(interval=0.5)

    if "virt_mem" in attrs:
        info["virtual_memory"] = psutil.virtual_memory()

    if "swap_mem" in attrs:
        info["swap_memory"] = psutil.virtual_memory()

    if "disk_io" in attrs:
        info["disk_io_counters"] = psutil.disk_io_counters()

    if "net_io" in attrs:
        info["net_io_counters"] = psutil.net_io_counters()
    
    info["timestamp"] = rfc3339(datetime.now())

    return simplejson.dumps(info)

if __name__ == "__main__": 

    attrs = ["cpu", "virt_mem", "swap_mem", "disk_io", "net_io"]
    print usages(attrs)
