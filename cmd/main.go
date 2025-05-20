package main

import (
	
	"mon-projet-go/server"

	"mon-projet-go/agent"
		
)



func main() {

	go server.Start() 
	
	go agent.Start()

	 select {}

}