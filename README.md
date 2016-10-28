### DMoni Overview

DMoni is an open source project to benchmark distributed applications and monitor
their performance.

In a cluster, a distributed application runs on many nodes/VMs; on each
node, it has serveral processes running. DMoni is able to
* measure the execution time of a distirbuted application lauched by it, and 
* monitor the reusource usages (CPU, memory, disk IO and network IO) of all the
processes running on different nodes of the application.

As a result, the collected performance data can be used for further performance analysis.

Currently, DMoni supports applications based Hadoop or Spark. It monitors both
processes of an applcation and of Hadoop or Spark, which is important
to have a thorough understanding of the application's performance.

### Concepts

DMoni has a master-slave architecture in a cluster. On each cluster node, there 
is a DMoni deamon running. The deamon can be run as two different roles, namely,
manager and agent. A cluster consists of one manager and many agents.

A manager is the master of a cluster and is in charge of
* Talking with clients to submit/kill an application and query the status of the application;
* Talking with agents to send instructions for launching, killing as well as monitoring applications;
* Maintaining DMoni cluster. It maintains a list of alive agents and deals with joining
and leaving of agents.

An agent act as a slave of manager, and is responsible for
* Launching/Killing an application (or starting/killing the main process of the application);
* Detecting processes of the application on the node where the agent is running;
* Monitoring resource usages of the processes;
* Notifying the manager when the application exits;
* Storing monitored data in ElasticSearch.

[Monica](https://gadget.ncsl.nist.gov:8081/lizhong/monica) is a CLI client for 
DMoni and is able to submit (or launch) and kill applicaitons, as well as query 
monitored performance data.

### Get Started

#### Prerequisites

* ElasticSearch: a document-oriented databse used to store benchmark data.

#### Installing

Step 1. Download the prebuilt package.
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

Step 2. Install snapshot package's dependencies

```
$ cd /tmp/dmoni/snapshot && ls 
LICENSE.txt  README.md  setup.py  snapshot
$ pip install .
```

step 3. Creat a path for DMoni

```
$ mkdir /usr/local/dmoni
$ cp -R /tmp/dmoni/dmoni /tmp/dmoni/snapshot/snapshot /usr/local/dmoni
```
The path created will be exported later, so that dmoni can find where the
snapshot pachage is.

Congratulation, the installation is done.

#### Run a local dmoni cluster

In this example, we are going to run a manger and two agents on a single node.
Since DMoni collects systems and processes' resource usages, it is needed to be
run with sudo previlege.

*Run dmoni manager*
```
$ sudo /usr/local/dmoni/dmoni manager --ds "http://192.168.0.3:9200"
2016/10/28 15:44:20 Dmoni Manager
```
The ```--ds``` option is the address of ElasticSearch.

*Run agent-0*
```
$ sudo env DMONIPATH="/usr/local/dmoni"  /usr/local/dmoni/dmoni agent \
  --id agent-0 --ip 192.168.0.6 --port 5301 \
  --mip 127.0.0.1 --ds "http://192.168.0.3:9200" 
2016/10/28 15:52:42 Dmoni Agent
2016/10/28 15:52:43 Say Hi to manager 127.0.0.1:5300
```
The options include the agent's info (id, ip and port), manager's address,
and data storage (ElasticSearch) 's address.

*Run agent-1*
```
$ sudo env DMONIPATH="/usr/local/dmoni"  /usr/local/dmoni/dmoni agent \
  --id agent-1 --ip 192.168.0.6 --port 5302 \
  --mip 127.0.0.1 --ds "http://192.168.0.3:9200" 
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
monitor them by using dmoni's client, [Monica](https://gadget.ncsl.nist.gov:8081/lizhong/monica).

### License
