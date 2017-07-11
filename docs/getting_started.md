## Getting Started

### Prerequisites

* ElasticSearch: a document-oriented databse used to store benchmark data.

### Installing

#### Step 1. Download the prebuilt package.
```
$ wget https://192.168.0.5:8080/swift/v1/dmoni/dmoni-0.9.0.tar.gz 
$ tar -xvf ./dmoni-0.9.0.tar.gz -C /tmp
$ cd /tmp/dmoni && ls
dmoni  README.md  snapshot
```
We can see that three files are included:
* dmoni: the binary file of DMoni;
* snapshot: python package used to obtain process and system's resource usages;
* README.md.

#### Step 2. Install snapshot package's dependencies

```
$ cd /tmp/dmoni/snapshot && ls 
LICENSE.txt  README.md  setup.py  snapshot
$ pip install .
```

#### step 3. Creat a path for DMoni

```
$ mkdir /usr/local/dmoni
$ cp -R /tmp/dmoni/dmoni /tmp/dmoni/snapshot/snapshot /usr/local/dmoni
```
The path created will be exported later, so that dmoni can find where the
snapshot pachage is.

Congratulation, the installation is done.

### Run a local dmoni cluster

In this example, we are going to run a manger and two agents on a single node.
Since DMoni collects systems and processes' resource usages, it is needed to be
run with sudo previlege.

#### Run dmoni manager
```
$ sudo /usr/local/dmoni/dmoni manager --storage "http://192.168.0.3:9200"
2016/10/28 15:44:20 Dmoni Manager
```
The ```--storage``` option is the address of ElasticSearch.

#### Run agent-0
```
$ sudo env DMONIPATH="/usr/local/dmoni"  /usr/local/dmoni/dmoni agent \
  --id agent-0 --ip 192.168.0.6 --port 5301 \
  --mip 127.0.0.1 --storage "http://192.168.0.3:9200" 
2016/10/28 15:52:42 Dmoni Agent
2016/10/28 15:52:43 Say Hi to manager 127.0.0.1:5300
```
The options include the agent's info (id, ip and port), manager's address,
and data storage (ElasticSearch) 's address.

#### Run agent-1
```
$ sudo env DMONIPATH="/usr/local/dmoni"  /usr/local/dmoni/dmoni agent \
  --id agent-1 --ip 192.168.0.6 --port 5302 \
  --mip 127.0.0.1 --storage "http://192.168.0.3:9200" 
2016/10/28 15:54:41 Dmoni Agent
2016/10/28 15:54:42 Say Hi to manager 127.0.0.1:5300
```

After creating the two agents, the manager's log looks like:
```
2016/10/28 15:52:43 SayHi from agent-0
2016/10/28 15:52:43 New agent agent-0 192.168.0.6:5301
2016/10/28 15:54:42 SayHi from agent-1
2016/10/28 15:54:42 New agent agent-1 192.168.0.6:5302
2016/10/28 15:54:43 SayHi from agent-0
2016/10/28 15:55:12 SayHi from agent-1
...
```

Congrats, a dmoni cluster is created. You can submit applicaitons and
monitor it.

### Submit and monitor an application

The `monica` subcommand is used to submit, kill and retieve performance
metrics of an application. The following is an example of launching a Spark 
applcation calculating Pi.

#### Submit the applicaiton
```
$ dmoni monica submit --manager 192.168.0.6 \
  --storage "http://192.168.0.3:9200" \
  --host 192.168.0.6 --frameworks spark --perf \
  /usr/local/spark/bin/run-example SparkPi 300
Application Id: a5da8a13486849e2b2f7317cc1362083
```

The Spark application's name is `/usr/local/spark/bin/run-example`. The `--manager`
and `--storage` options specify the address of dmoni's manager and storage.
The `--host` option tells where the application should be launched, in this example,
it is on host ```192.168.0.6```. An application can be launched on any host with a
dmoni agent running. The ``--perf`` option enables monitoring the application as
well as underlying frameworks specified by ```--frameworks``` option.
It returns an ID for the launched application, which can be used to obtain info
of and kill the application afterwards.

#### Obtain basic info of an application

```
$ dmoni monica app --manager 192.168.0.6 \
  --storage "http://192.168.0.3:9200" \
  --id a5da8a13486849e2b2f7317cc1362083
Using config file: /home/lnz5/.monica.yaml
{
        "app_id" : "a5da8a13486849e2b2f7317cc1362083",
        "args" : [ "SparkPi", "300" ],
        "entry_node" : "192.168.0.6",
        "exec" : "/usr/local/spark/bin/run-example",
        "start_at" : "2016-10-24T16:46:33-04:00",
        "timestamp" : "2016-10-24T16:46:33-04:00",
        "end_at" : "2016-10-24T16:46:45-04:00",
        "stdout" : "Pi is roughly 3.142029038067635\n",
        "stderr" : "16/10/24 16:46:35 INFO spark.SparkContext: Running ..."
}
```
It returns the executed command (```exec``` and ```args```), start time (```start_at```)
, end time (```end_at```), ```stdout``` and ```stderr``` of the applicaiton's main 
process in JSON format.

#### Obtain measured application's performance data

```
$ dmoni monica app  --manager 192.168.0.6 \
  --storage "http://192.168.0.3:9200" \
  --id a5da8a13486849e2b2f7317cc1362083 --perf
{
        "app_id" : "a5da8a13486849e2b2f7317cc1362083",
        "cmdline" : [ "/usr/lib/jvm/java-7-openjdk-amd64/bin/java", "-cp", "/usr/local/spark/conf/:/usr/local/spark/jars/*:/usr/local/hadoop/etc/hadoop/", "-Xmx1g", "-XX:MaxPermSize=256m", "org.apache.spark.deploy.history.HistoryServer" ],
        "cpu_percent" : 0,
        "cpu_times" : {
          "children_system" : 0.07,
          "children_user" : 0.2,
          "system" : 835.86,
          "user" : 2050.75
        },
        "exe" : "/usr/lib/jvm/java-7-openjdk-amd64/jre/bin/java",
        "memory_info" : {
          "data" : 3.4501632E9,
          "dirty" : 0,
          "lib" : 0,
          "rss" : 5.28736256E8,
          "shared" : 2.59072E7,
          "text" : 4096,
          "vms" : 3.551555584E9
        },
        "memory_percent" : 1.604808333277389,
        "name" : "java",
        "node" : "192.168.0.6",
        "num_threads" : 29,
        "pid" : 5866,
        "timestamp" : "2016-10-24T16:46:44-04:00",
        "username" : "hduser"
}
...
```

The ```--perf``` option asks to return resource usages of the application's 
and underliying frameworks' processes during the application's lifecycle. Each
measurement corresponds to a single process's resource usage at a moment, 
and consists of a timestamp, node IP, process info, and CPU usage, memory usage,
and etc.

#### Obtain measured systems' performance data
```
$ dmoni monica app  --manager 192.168.0.6 \
  --storage "http://192.168.0.3:9200" \
  --id a5da8a13486849e2b2f7317cc1362083 --sys-perf
{
        "cpu_count" : 8,
        "cpu_percent" : 0.2,
        "cpu_times" : {
          "guest" : 0,
          "guest_nice" : 0,
          "idle" : 4685357.77,
          "iowait" : 30579.71,
          "irq" : 2.36,
          "nice" : 65.88,
          "softirq" : 345.27,
          "steal" : 0,
          "system" : 9237.89,
          "user" : 40133.14
        },
        "disk_io_counters" : {
          "busy_time" : 3.3547572E7,
          "read_bytes" : 7.449767936E9,
          "read_count" : 225324,
          "read_merged_count" : 3,
          "read_time" : 3012232.0,
          "write_bytes" : 2.97480179712E11,
          "write_count" : 5.2760674E7,
          "write_merged_count" : 6475402.0,
          "write_time" : 1.26968968E8
        },
        "net_io_counters" : {
          "bytes_recv" : 4.8797874859E10,
          "bytes_sent" : 7.8962352246E10,
          "dropin" : 0,
          "dropout" : 0,
          "errin" : 0,
          "errout" : 0,
          "packets_recv" : 9.061063E7,
          "packets_sent" : 5.9475744E7
        },
        "node" : "192.168.0.6",
        "swap_memory" : {
          "active" : 2.5726693376E10,
          "available" : 2.861832192E10,
          "buffers" : 5.17517312E8,
          "cached" : 2.3322689536E10,
          "free" : 4.778115072E9,
          "inactive" : 1.484660736E9,
          "percent" : 13.1,
          "shared" : 6025216.0,
          "total" : 3.2947003392E10,
          "used" : 2.816888832E10
        },
        "timestamp" : "2016-10-24T16:48:30-04:00",
        "virtual_memory" : {
          "active" : 2.5726693376E10,
          "available" : 2.861832192E10,
          "buffers" : 5.17517312E8,
          "cached" : 2.3322689536E10,
          "free" : 4.778115072E9,
          "inactive" : 1.484660736E9,
          "percent" : 13.1,
          "shared" : 6025216.0,
          "total" : 3.2947003392E10,
          "used" : 2.816888832E10
        }
}
...
```
The ```--sys-perf``` option asks to return resource usages of systems where the
application has/had been running. Each measurement includes a timestamp, CUP
usage, memory usage, disk IO counter and network IO counter.

#### Kill an application
```
$ dmoni monica kill --id a5da8a13486849e2b2f7317cc1362083
```

## License
