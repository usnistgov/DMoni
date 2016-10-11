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
* Notifying the manager when the application exits.

### Get Started

#### Installing

#### Run a local DMoni cluster

### License