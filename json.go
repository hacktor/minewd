package main

import (
	"encoding/json"
	"log"
	"time"
)

func (p packet) analyzeJSON(c chan dbRecord) {

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
			db.batt = 0
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
