# Snapshot

Snapshot is for monitoring a given process' utilization of system resources,
including CPU, memory and io. Specifically,

* process basic info: name, commandline, username;
* cpu: cpu times, cpu percentage, number of threads;
* memory: memory usage, memory percentage;
* io: the number of read and write operations and the amount of bytes read and written;
* descendant processes.

Since Spapshot uses psutil package to gather
process utilization, the detailed explaination about output data can be found at 
[psutil documentation](http://pythonhosted.org/psutil/).

## Installation

```bash
$ pip install . && cd ./snapshot
```

## Usage
```
$ python monitor.py -h
usage: monitor.py [-h] [-i [INTERVAL]] [-n [N]] [-o [Filename]]
                  [--attrs [ATTRS]] [--desc]
                  pid

positional arguments:
  pid              Process id

optional arguments:
  -h, --help       show this help message and exit
  -i [INTERVAL]    interval (seconds) between two collections. Defualt vaule:
                   3
  -n [N]           number of collections. If n == 0, collecting info until the
                   process exits. Default value: 0
  -o [Filename]    output file name. Defualt: stdout
  --attrs [ATTRS]  list of attributes to be collected. Including cpu, mem,
                   mem_full and io. Default value: cpu,mem
  --desc           Collecting the info of the process' descendants

```
### Examples
#### Monitoring a process until it exits
```bash
$ python monitor.py 7789
{"username": "lnz5", "num_threads": 61, "exe": "/usr/lib/firefox/firefox", "cpu_times": [23166.18, 1656.22, 786.28, 237.08], "name": "firefox", "cpu_percent": 28.0, "timestamp": "2016-08-19 15:14:10.398602", "memory_percent": 12.027069001487353, "pid": 7789, "cmdline": ["/usr/lib/firefox/firefox"], "memory_info": [2009460736, 3560611840, 224808960, 118784, 0, 2494603264, 0]}
{"username": "lnz5", "num_threads": 61, "exe": "/usr/lib/firefox/firefox", "cpu_times": [23167.68, 1656.27, 786.28, 237.08], "name": "firefox", "cpu_percent": 68.9, "timestamp": "2016-08-19 15:14:14.404428", "memory_percent": 11.776471865923915, "pid": 7789, "cmdline": ["/usr/lib/firefox/firefox"], "memory_info": [1967591424, 3561005056, 224808960, 118784, 0, 2494996480, 0]}
...
```
Where the process id 7789 is given and the outputs are in json format. By
default, cpu and memory info is collected.

#### Collecting a process's utilization 5 times with interval 30 seconds
```bash
$ python monitor.py -n 5 -i 30 7788
```

#### Monitoring a process and its descendant processes
```bash
$ python monitor.py --desc 7788
```
