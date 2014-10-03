Backends Updater
=====================

This demon do following operation in order:

1. Get list of running containers and identify app containers using specified regex pattern
2. Generate loadbalancer(nginx) config file with found app containers as backends(upstream servers)
3. Listen to docker events for any 'start' or 'stop' events.

    If any app container is created or stopped, it then add or remove this entry respectively from load balancer configuration, update configuration file and restart nginx . Before updating config file, it takes backup of old configuration file
    
    
Installation
------------

### Pre-requisite

- golang (preferrably ~ 1.3)


### Build binary

```
#Steps to build
git clone project
go build main.go
mv main backendUpdater

```

 
### Usage

```
D:\tmp\go\tryOne>main --help
Usage:
  -bkp_dir="": file backup directory(Optional)
  -config="": Configuration file path(*required) eg. /etc/nginx/sites-enabled/default
  -dockerAddr="0.0.0.0:4243": docker bind addresss(Optional)
  -heartbeat=30s: heartbeat interval for containers check. eg 30s , 5m, 30m (Optional)
  -host="127.0.0.1": hostname or IP to attach to containers(optional)
  -service="": Service Pattern to track(*required)
  -since=1412355983: docker events from whence(optional)
  -template="loadbalancer.conf": template file(required)

```

* **bkp_dir** : Directory to store old config files
* **dockerAdd** :  Address to listen for docker remote api
* **exportfile** : Path to export config file (usually sites-enabled folder)
* **hearteat** : Interval to wait before updating file
* **host** : IP Address to attach in backend server list . PORT will be populated from container public port information
* **service** : Regex service pattern to find app containers
* **since** : To track events since XXX
* **template** : load balancer config template


### Example

```
main -dockerAddr http://199.127.219.76:4243  -service "dailyReport[0-9]*.stackepress.com" -config "default"
```


System V init script template
=============================

A simple template for init scripts that provide the start, stop,
restart and status commands.

Getting started
---------------

Copy _backendUpdater_ to /etc/init.d and rename it to something
meaningful. Then edit the script and enter that name after _Provides:_
(between _### BEGIN INIT INFO_ and _### END INIT INFO_).

Now set the following three variables in the script:

### dir ###

The working directory of your process.

### user ###

The user that should execute the command.

### cmd ###

The command line to start the process.

Here's an example for an app called
[backendUpdater](http://backendUpdater.ubercode.de):

    dir="/var/apps/backendUpdater"
    user="node"
    cmd='/usr/bin/backendUpdater -dockerAddr http://127.0.0.1:4243  -service "dailyReport[0-9]*.stackepress.com" -config "default" '

Script usage
------------

### Start ###

Starts the app.

    /etc/init.d/backendUpdater start

### Stop ###

Stops the app.

    /etc/init.d/backendUpdater stop

### Restart ###

Restarts the app.

    /etc/init.d/backendUpdater restart

### Status ###

Tells you whether the app is running. Exits with _0_ if it is and _1_
otherwise.

    /etc/init.d/backendUpdater status

Logging
-------

By default, standard output goes to _/var/log/scriptname.log_ and
error output to _/var/log/scriptname.err_. If you're not happy with
that, change the variables `stdout_log` and `stderr_log`.


How code works?
--------------

Goals before writing code

- Keep update( config file) and restart(nginx ) operation to minimum

    i.e If many docker events are triggered in short interval, then update only after populating results from all events. This update interval is customizale with `heartbeat` command arguments. 
	- It'll wait for specified `heartbeat` interval. In this all docker events will be queued. After that each event is processed
	- Summary of all queued event is populated into Final Updated Container list
	- If app container list is changed, then only loadbalancer config file is updated and nginx restarted, else nothing happened


RoadMap
--------------

- Backends healthcheck 
- Remove unhealthy backends
 