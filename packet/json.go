package main

import (
	"encoding/json"
	"log"
	"net"
	"time"
)

type jsonpacket struct {
	content []byte
	remAddr string
	reqLen  int
	boxID   string
}

func handleConn(conn net.Conn, c chan dbRecord) {

	defer conn.Close()

	// Make a buffer to hold incoming data.
	var p jsonpacket
	p.content = make([]byte, 16384)
	var err error

	// Read the incoming connection into the buffer.
	p.remAddr = conn.RemoteAddr().String()
	p.reqLen, err = conn.Read(p.content)
	if err != nil {
		log.Println(p.remAddr, "closed connection:", err.Error())
		return
	}
	log.Println("Incoming from:", p.remAddr)

	// Analyze packet
	p.analyzeBLE(c)

	// Send a response back
	_, err = conn.Write([]byte("Message received: " + string(p.content[:p.reqLen])))
	if err != nil {
		log.Println("Error wrinting to", p.remAddr, ":", err.Error())
	}
}

func (p jsonpacket) analyzeBLE(c chan dbRecord) {

	// p.content is json data
	var tags []map[string]interface{}

	err := json.Unmarshal([]byte(p.content[:p.reqLen]), &tags)
	if err != nil {
		log.Println("json error:", err.Error())
		return
	}

	tnow := sqltime(time.Now().Unix())

	for key, tag := range tags {

		// first tag is the gateway, extract boxID
		if key == 0 {
			if str, ok := tag["mac"].(string); ok {
				p.boxID = str
			}
			continue
		}

		var db dbRecord
		db.dateTime = tnow
		db.boxID = p.boxID

		if str, ok := tag["mac"].(string); ok {
			db.tagID = str
		} else {
			log.Println("mac failed:", tag["mac"])
			continue
		}
		if num, ok := tag["rssi"].(float64); ok {
			db.rssi = int(num)
		} else {
			db.rssi = -127
		}
		if num, ok := tag["battery"].(float64); ok {
			db.batt = int(num)
		} else {
			db.rssi = -127
		}
		if str, ok := tag["rawData"].(string); ok {
			db.data = str
		} else {
			db.data = ""
		}

		// send dbRecord to database through channel
		c <- db
	}
}
