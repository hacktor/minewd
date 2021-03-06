package main

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func (p packet) analyzeBLE(c chan dbRecord) {

    var nrBLE uint16 = 0
	if p.content[0] == 187 && p.content[p.reqLen-1] == 221 && p.reqLen > 22 {

		p.boxID = hex.EncodeToString(p.content[6:12])
		nrBLE = binary.BigEndian.Uint16(p.content[12:14])
	} else {
		log.Println("Ignoring corrupted packet")
		return
	}

	ble := p.content[14:]

	for i := nrBLE; i > 0; i-- {

		var d uint16
		tnow := sqltime(time.Now().Unix())

		tagID := hex.EncodeToString(ble[0:6])
		nrBLEdata := binary.BigEndian.Uint16(ble[6:8])
		if len(ble) > 8 {
			ble = ble[8:]
		} else {
			return
		}
		for d = 0; d < nrBLEdata; d++ {

			rawLen := int(ble[0])
			if len(ble) < rawLen+2 {
				log.Println("Packet length exceeded")
				return
			}
			var db dbRecord
			db.dateTime = tnow
			db.boxID = p.boxID
			db.tagID = tagID
			if rawLen != 0 {
				db.data = hex.EncodeToString(ble[1 : 1+rawLen])
			}
			db.rssi = int(ble[rawLen+2])
			// we don't get the battery status in binary mode, so send fake value 100%
			db.batt = 100

			// send dbRecord to database through channel
			c <- db

			if len(ble) > rawLen+3 {
				ble = ble[rawLen+3:]
			} else {
				return
			}
		}
	}
}
