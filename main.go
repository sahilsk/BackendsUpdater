/*
Copyright 2014 Stackexpress.com
----------------------------------
Author: Sonu K. Meena
Email: sonu.k.meena@stackexpress.com
---------------------------------
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net/http"
	"flag"
	"log"
	"encoding/json"
	"regexp"
	"os"
	"os/exec"
	"text/template"
	"io"
	"time"
	"strconv"
	"strings"
	"errors"
)
	
//loadbalancer template
const templateFile = "loadbalancer.conf.tmpl"
const serverRestartCMD = "/etc/init.d/nginx restart"
	
/************ Command line input:
*
*/

var (
// Commad line args
	hostname, err = os.Hostname()
	cmd_dockerAddress = flag.String("dockerAddr", "127.0.0.1:4243", "docker bind addresss(Optional)")
	cmd_since = flag.Int64("since", time.Now().Unix(), "docker events from whence(optional)")
	cmd_host = flag.String("host", "127.0.0.1", "hostname or IP to attach to containers(optional)")
	cmd_serviceRegex = flag.String("service", "", "Service Pattern to track eg. app[0-9]*.stackexpress.com (*required)")
	cmd_heartbeat = flag.Duration("heartbeat", 30*time.Second, "heartbeat interval for containers check. eg 30s , 5m, 30m (Optional)")
	cmd_bkpDir = flag.String("bkp_dir", "", "file backup directory(Optional)")
	cmd_template = flag.String("template", templateFile, "template file(*required)")
	cmd_configFile = flag.String("config", "", "Configuration file path eg. /etc/nginx/sites-enabled/default (*required)")

	
	// Global ContainerArray
	containerArray []Container
)


type ContainerListResp []struct {
		Command, Id, Image, Status string
		Names []string
		Ports []struct{
			IP , Type string
			PrivatePort, PublicPort uint
		}
}
type EventResp struct{
	Status, Id, From string
	Time uint
}
type Event struct{
	Status, Id, From string
	Time uint
}

type ContainerResp struct{
	Created, Driver, ExecDriver string
	HostConfig struct{
		PortBindings map[string][]struct{
		HostIp, HostPort string
		}
	}
}

type Container struct{
	Id string
	Name string
	Ip string
	Port uint
}

func getFullURL( cmd string) string {
	
	var (
		containers = regexp.MustCompile(`^/containers/json*`)
		container = regexp.MustCompile(`^/containers/[a-z0-9]*/json*`)
		events = regexp.MustCompile(`^/events/*`)
		version = regexp.MustCompile(`^/version*`)
	)

	switch {
	case containers.MatchString(cmd):
		return fmt.Sprintf( "%s/%s", *cmd_dockerAddress, "containers/json")
	case container.MatchString(cmd):
		return fmt.Sprintf( "%s/%s", *cmd_dockerAddress, cmd)	
	case events.MatchString(cmd):
		return fmt.Sprintf( "%s/%s%d", *cmd_dockerAddress, "events?since=", *cmd_since)
	case version.MatchString(cmd):
		return fmt.Sprintf( "%s/%s", *cmd_dockerAddress, "/version")		
	default:
		return *cmd_dockerAddress
	}
	
}

func perror( err error){
	if err != nil{
		panic(err)
	}
}

func inspectContainer( Id string){
	
	resp, err := http.Get( getFullURL("/containers/" + Id + "/json") )
	perror(err)
	var response ContainerResp
	err = json.NewDecoder( resp.Body).Decode( &response)
	
	fmt.Println(getFullURL("/containers/" + Id + "/json"))
//	fmt.Println(response)
	fmt.Println(response.HostConfig.PortBindings)

	for k, v := range( response.HostConfig.PortBindings) {
		fmt.Printf("%s => %s\n", k, v[0].HostPort)
	}
	


}

//Get containers whose name matching cmd_serviceRegex
func getMatchedContainers( url string) []Container {
	
	log.Println("Get: " + url )
	resp, err := http.Get(url)
	
	defer resp.Body.Close()
	perror(err)

	
	var watchedServiceRegex = regexp.MustCompile(*cmd_serviceRegex)
	
	var  response ContainerListResp
	err = json.NewDecoder( resp.Body).Decode( &response)
	
	
	var cArray = []Container{};
	
	for _,  item := range(response) {
		
	//	fmt.Printf( "%s \t| %s \t| %s \t| %d:%d \n", string(item.Command[:24]), string(item.Id[:12]), item.Status, item.Ports[0].PublicPort,item.Ports[0].PrivatePort)
		
		for _, name := range(item.Names) {
			
			if watchedServiceRegex.MatchString(name){
				var container Container = Container{ item.Id, name, *cmd_host, item.Ports[0].PublicPort}
				cArray = append(cArray, container)
				break;
			}
		}		
	}
	

	return cArray
	
}

func monitorEvents( url string, queue chan Event){
	
	
	log.Println("Get: " + url )
	resp, err := http.Get(url)
	perror(err)
	
	defer resp.Body.Close()
	
	
	dec := json.NewDecoder( resp.Body)
	
	for {
		var response Event
		err = dec.Decode( &response)
		if err == io.EOF{
			break
		} else if err != nil{
			log.Fatal(err)
		}
		
		log.Printf( "%s \t| %s \t| %s \t| %d \n", string(response.Id[:12]), response.Status, response.From, response.Time)
		queue <- response
			
	}

}


//go routine to udpate file after every 5 minutes
func restartNginx() error {
	
	var serverRestart = strings.Split( serverRestartCMD, " ")
	
	out, err := exec.Command(serverRestart[0], serverRestart[1:]... ).Output()
	if err != nil {
		return err
	}
	log.Printf( "%s", out)
	if strings.Contains( string(out), "fail"){
		err = errors.New("Failed to run command")
	}
	return err;
	
}


func updateLoadbalancer(){
		
			
		//Template 
		templ_loadbalancerConfig := template.New( templateFile )
		templ_loadbalancerConfig,err = templ_loadbalancerConfig.ParseFiles(templateFile)
		perror(err)	
		
		
		//Take backup of old file
		if len(*cmd_bkpDir) > 0 {
			err = os.Rename(*cmd_configFile, *cmd_bkpDir +"/" +strconv.FormatInt( time.Now().UnixNano() / int64(time.Millisecond), 10 )  )
			//err = os.Rename(*cmd_configFile, "abc" )
			perror(err)
		}
		
		
		log.Println("======== Updating configuration file ");
		//Create nging config file
		f, err := os.Create( *cmd_configFile)
		perror(err)
		
		defer f.Close()		
		
		//Update file
		err = templ_loadbalancerConfig.Execute(  f, map[string] interface{} { "containers" : containerArray, "LastModified"  : time.Now() } )
		perror(err)
		
		log.Println("======== Restarting Nginx ")
		//Restart Nginx
		err = restartNginx()	
		perror(err)
}


func execEvent( event Event) bool{
	var shouldUpdate = false
	if event.Status == "start" || event.Status == "create"{
	
		log.Println("======== CONTAINER STARTED ==========\n")
		var containerArray2 = getMatchedContainers( getFullURL("/containers/json") )
		
		var isAppContainer = false
		for _, k := range(containerArray2) {
			if k.Id == event.Id {
				isAppContainer = true
				break
			}
		}
		if isAppContainer == true {
			var hasEntry = false
			for _, k := range(containerArray) {
				if k.Id == event.Id {
					hasEntry = true
					break
				}
			}
			if hasEntry == false{
				containerArray = containerArray2
				shouldUpdate = true
			}
		}
		
	}else if event.Status == "die" || event.Status == "stop" {
		log.Println("======== CONTAINER STOPPED ==========\n")
		for i, item:= range(containerArray){
			if item.Id == event.Id{
				containerArray[i] = containerArray[ len(containerArray) -1]
				containerArray = containerArray[ 0:len(containerArray) -1]
			}
		}
	}
	
	return shouldUpdate
}

func eventConsumer(queue chan Event){
	
	for {
		//sleep for heartbeat interval
        time.Sleep( *cmd_heartbeat )
		//After waking up  clear the queue and update the config
		if len(queue) > 0 {
			var shouldUpdate = false
			for i:=0;i < len(queue);i++{
				event := <- queue
				shouldUpdate = shouldUpdate || execEvent( event)
			}
			if shouldUpdate{
				updateLoadbalancer()
			}
		}else {
			log.Println("====== No file updated ")
		}
    }
}

func test() error{
	//resp, err := http.Get( getFullURL("/version") )
	
	return err
}

func Usage() {
        fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
        flag.PrintDefaults()
}

func main(){
	flag.Parse()
	var _ = fmt.Println
	var _ = http.StatusOK

	//check required args
	if len(*cmd_serviceRegex) < 1 || len(*cmd_configFile) < 1 || len(*cmd_template) < 1 {
		Usage()
		return
	}

	queue := make(chan Event, 100 )
	
	go eventConsumer( queue)
	
	//Update config with running App container
	containerArray := getMatchedContainers( getFullURL("/containers/json") )
	updateLoadbalancer()
	
	log.Println( containerArray)
	
	//Infinite Monitoring
	monitorEvents( getFullURL("/events?since=1122")	, queue)
	
}