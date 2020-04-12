package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/ini.v1"
)

type dbRecord struct {
	tagID string
	boxID string
	rssi  int8
	data  string
}

type tag struct {
	tagID     string
	boxID     string
	rssi      int
	nrBLEdata uint16
}

type packet struct {
	datal uint32
	boxID string
	nrBLE uint16
	tags  []tag
}

var db *sql.DB // global variables to share between main and the handlers
var stmt *sql.Stmt

func main() {

	cfg, err := ini.Load("minewd.ini")
	if err != nil {
		log.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	// start database goroutine
	c := make(chan dbRecord)
	go database(cfg, c)

	// start listening
	ln, err := net.Listen("tcp", ":"+cfg.Section("ontvanger").Key("port").String())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer ln.Close()
	log.Println("Listening on port " + cfg.Section("ontvanger").Key("port").String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}

		go handleConn(conn, c)
	}
}

func database(cfg *ini.File, c chan dbRecord) {

	// connect to database if ontvanger->dbsav is defined
	if cfg.Section("ontvanger").HasKey("dbsav") &&
		cfg.Section("database").Key("type").String() == "mysql" {

		// Build connection string
		connStr := cfg.Section("database").Key("user").String() + ":" + cfg.Section("database").Key("pass").String() +
			"@/" + cfg.Section("database").Key("db").String() + "?charset=utf8"

		// Create connection pool
		db, err := sql.Open("mysql", connStr)
		if err != nil {
			log.Println("Error opening database->file", cfg.Section("database").Key("file").String())
			panic(err)
		}

		stmt, err = db.Prepare("INSERT INTO minew(boxID, tagID, rssi, batt) values(?,?,?,?)")
		if err != nil {
			panic(err)
		}

		log.Println("Connected to database", cfg.Section("database").Key("db").String())
	}
	for r := range c {
		stmt.Exec(r.boxID, r.tagID, r.rssi, string(r.data))
	}
}

func handleConn(conn net.Conn, c chan dbRecord) {

	defer conn.Close()

	// Make a buffer to hold incoming data.
	buf := make([]byte, 4096)
	var p packet

	// Read the incoming connection into the buffer.
	remAddr := conn.RemoteAddr().String()
	reqLen, err := conn.Read(buf)
	if err != nil {
		log.Println(remAddr, "closed connection:", err.Error())
		return
	}
	// log.Println("From", remAddr, ":", buf[:reqLen])

	// Analyze packet
	p.analyzeBLE(buf, reqLen, c)

	// Send a response back
	_, err = conn.Write([]byte("Message received: " + string(buf)))
	if err != nil {
		log.Println("Error wrinting to", remAddr, ":", err.Error())
	}
}

func (p packet) analyzeBLE(buf []byte, reqLen int, c chan dbRecord) {

	if buf[0] == 187 && buf[reqLen-1] == 221 {

		p.datal = binary.BigEndian.Uint32(buf[2:6])
		p.boxID = hex.EncodeToString(buf[6:12])
		p.nrBLE = binary.BigEndian.Uint16(buf[12:14])
	}

	ble := buf[14 : reqLen-1]

	for i := p.nrBLE; i > 0; i-- {

		var t tag
		var d uint16

		t.boxID = p.boxID
		t.tagID = hex.EncodeToString(ble[0:6])
		t.nrBLEdata = binary.BigEndian.Uint16(ble[6:8])
		ble = ble[8:]
		for d = 0; d < t.nrBLEdata; d++ {
			// meestal 1
			// schrijf naar channel
		}

	}
	log.Println("datal:", p.datal)
	log.Println("BoxID:", p.boxID)
	log.Println("nrBLE:", p.nrBLE)
}

func sqltime(ts int64) string {

	// return timestamp formatted for mssql
	const sqlts = "2006-01-02 15:04:05"
	loc, _ := time.LoadLocation("UTC")
	t := time.Unix(ts, 0).In(loc)
	return t.Format(sqlts)
}
