import psutil
from datetime import datetime
from rfc3339 import rfc3339

def shot(procId, attrs, descendant=False):

    '''Collect system resource usage info of a given process.
    The information is obtained through psutil package.
    '''

    proc = psutil.Process(procId)
    pic = {}

    if proc.is_running() and proc.status()!=psutil.STATUS_ZOMBIE:
        # cpu_percent is obtained seperately, because it is calculated
        # based on the difference between process cpu times and system
        # times.
        pic.update(proc.as_dict(attrs=attrs))
        if "cpu_percent" in attrs:
            pic["cpu_percent"] = proc.cpu_percent(interval=1)
        pic["timestamp"] = rfc3339(datetime.now())
        
        if descendant:
            # Collect descendant prcesses' info
            children = proc.children()
            if len(children) > 0:
                pic["children"] = []
                for c in children:
                    cPic = shot(c.pid, attrs, descendant)
                    pic["children"].append(cPic)
    return pic
           

def print_pic(pic, level):
    print "----"*level, pic["pid"], pic["name"]
    if "children" in pic:
        for c in pic["children"]:
            print_pic(c, level+1)

