## DMoni Overview

DMoni is an open source project to benchmark distributed applications and monitor their performance.

In a cluster, a distributed application runs on many nodes/VMs; on each node, it has several processes running. DMoni is able to
* measure the execution time of a distributed applications lauched by it, and 
* monitor the resource usages (CPU, memory, disk IO and network IO) of all the processes running on different nodes of the application.

As a result, the collected performance data can be used for further performance analysis.

Currently, DMoni supports applications based on [Hadoop](http://hadoop.apache.org/) or [Spark](https://spark.apache.org/). It monitors both the processes of an application and of Hadoop or Spark, allowing users to have a thorough understanding of the application's performance.

## Concepts

DMoni has a *master*-*worker* architecture in a cluster. 
Each cluster node has its own DMoni daemon running.
The daemon can be run as two different roles; *manager* and *agent*. A cluster consists of one *manager* and many *agents*.

A *manager* is the *master* of a cluster and is in charge of
* Talking with clients to submit/kill an application and query the status of an application;
* Talking with agents to send instructions for launching, killing as well as monitoring applications;
* Maintaining the DMoni cluster; it maintains a list of live agents and deals with the joining and leaving of agents.

An *agent* act as a *worker* for the *manager*, and is responsible for
* Launching/Killing an application (or starting/killing the main process of the application);
* Detecting processes of the application on the node where the agent is running;
* Monitoring processes resources usage;
* Notifying the manager when the application exits;
* Storing monitored data in [ElasticSearch](https://www.elastic.co/).

Using DMoni's command line interface, users can submit, kill as well as retrieve monitored performance metrics of an application.

## To start using DMoni

See our [getting started](/docs/getting_started.md) documentation.

## Support

If you have any question about DMoni or have a problem using it, please email [ems_poc@nist.gov](mailto:ems_poc@nist.gov).

## Contributors

* Lizhong Zhang ✻
* Martial Michel ❖ Project Lead

( ❖ NIST / ✻ Guest Researcher )

## Disclaimer

Certain commercial entities, equipment, or materials may be identified in this document in order to describe an experimental procedure or concept adequately. Such identification is not intended to imply recommendation or endorsement by the National Institute of Standards and Technology, nor is it intended to imply that the entities, materials, or equipment mentioned are necessarily the best available for the purpose.
All copyrights and trademarks are properties of their respective owners.