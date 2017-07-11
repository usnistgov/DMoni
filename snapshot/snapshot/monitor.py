import sys
import psutil
import time
import codecs
import simplejson
import io
import argparse
from common import shot

# Mapping self-defined attributes to psutil's attributes
attr_map = {
    'cpu': ['num_threads', 'cpu_times', 'cpu_percent'],
    'mem': ['memory_percent', 'memory_info'],
    'mem_full': ['memory_percent', 'memory_full_info'],
    'io': ['io_counters']
}

def monitor(pid, outFile, interval, n, attrs, descendant=False):

    """Monitoring a process' system resource usage periodly.
    The info collected are exported in JSON format.
    """

    i = 0
    # If n == 0 then monitor the process until it exits;
    # If n >= 1 then collect process info for n times.
    while n == 0 or i < n:
        try:
            pic = shot.shot(pid, attrs, descendant)
            if pic == {}: # Process has exited
                break

            if "memory_full_info" in pic:
                # rename memory_full_info as memory_info
                pic["memory_info"] = pic["memory_full_info"]
                del pic["memory_full_info"]

            data = simplejson.dumps(pic)
            outFile.write(data + "\n")
            i = i + 1
            if i < n:
                time.sleep(interval)
        except psutil.NoSuchProcess, psutil.AccessDenied:
            sys.exit("Error: No process %d or access denied." %pid)

            
def parse_args():
    
    '''Parse command line arguments
    '''

    parser = argparse.ArgumentParser()
    parser.add_argument('pid', type=int, help='Process id')
    parser.add_argument('-i', metavar='INTERVAL', nargs='?', type=int, default=3,
        help='interval (seconds) between two collections. Defualt vaule: 3')
    parser.add_argument('-n', nargs='?', type=int, default=0, 
        help='number of collections. If n == 0, collecting info until the process exits. Default value: 0')
    parser.add_argument('-o', nargs='?', type=str, metavar="Filename",
        help='output file name. Defualt: stdout')
    parser.add_argument( '--attrs', nargs='?', type=str, default="cpu,mem",
        help='list of attributes to be collected . Including cpu, mem, mem_full and io. Default value: cpu,mem')
    parser.add_argument( '--desc', action='store_true', help='Collecting the info of the process\' descendants')

    args = parser.parse_args()

    # validate the attributes
    attrs = []
    for a in args.attrs.split(','):
        if a not in attr_map.keys():
            parser.print_usage()
            print >>sys.stderr, "error: no attribute %s" %a
            sys.exit(2)
        else:
            attrs.append(a)
    args.attrs = attrs

    return args

if __name__ == "__main__":

    args = parse_args()

    # map self-defined attributes to psutil's attributes
    attrs = ["pid", "name", "username", "exe", "cmdline"]
    for a in args.attrs:
        attrs = attrs + attr_map[a]

    # If output file is given, use sys.stdout
    if args.o != None:
        f = codecs.open(args.o, "w", encoding="utf-8")
    else:
        f = sys.stdout

    monitor(args.pid, f, args.i, args.n, attrs, args.desc)

    f.close()

