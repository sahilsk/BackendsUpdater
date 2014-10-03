Backends Updater
=====================

This demon do following operation in order:

1. Get list of running containers and identify app containers using specified regex pattern
2. Generate loadbalancer(nginx) config file with found app containers as backends(upstream servers)
3. Listen to docker events for any 'start' or 'stop' events.

    If any app container is created or stopped, it then add or removes entry respectively from loadbalancer config and restart it. Before updating config file, it takes backup of old config
    
    
Installation
------------

### Pre-requisite

- golang (preferrably ~ 1.3)


### Build binary

```
#Steps to build
git clone project
go build main.go

```

 
### Run Arguments

```
D:\tmp\go\tryOne>main --help
Usage of main:
  -bkp_dir="./": file backup directory
  -dockerAddr="http://199.127.219.76:4243": docker bind addresss
  -exportfile="default": Export file path
  -heartbeat=30s: heartbeat interval for containers check
  -host="DRAGONAIDER": hostname
  -service="dailyreport[0-9]*.stackexpress.com": Service Pattern to track
  -since=1412342469: docker events from whence
  -template="loadbalancer.conf": template file

```

* **bkp_dir** : Directory to store old config files
* **dockerAdd** :  Address to listen for docker remote api
* **exportfile** : Path to export config file (usually sites-enabled folder)
* **hearteat** : Interval to wait before updating file
* **host** : IP Address to attach in backend server list . PORT will be populated from container public port information
* **service** : Regex service pattern to find app containers
* **since** : To track events since XXX
* **template** : load balancer config template


How code works?
--------------

Goals before writing code

- Keep update( config file) and restart(nginx ) operation to minimum

    i.e If many docker events are triggered in short interval, then update only after populating results from all events. This update interval is customizale with `heartbeat` command arguments. 
	- It'll wait for specified `heartbeat` interval. In this all docker events will be queued. After that each event is processed
	- Summary of all queued event is populated into Final Updated Container list
	- If app container list is changed, then only loadbalancer config file is updated and nginx restarted, else nothing happened



 