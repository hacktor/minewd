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
    dateTime string
    tagID    string
    boxID    string
    rssi     int8
    data     string
}

type packet struct {
    content []byte
    datal   uint32
    remAddr string
    reqLen  int
    boxID   string
    nrBLE   uint16
}

// global variables to share between main and the handlers
var db *sql.DB
var stmt *sql.Stmt
var c = make(chan dbRecord, 64)

func main() {

    // load configuration
    cfg, err := ini.Load("minewd.ini")
    if err != nil {
        log.Printf("Fail to read file: %v", err)
        os.Exit(1)
    }

    // start database goroutine
    go database(cfg)

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

        go handleConn(conn)
    }
}

func database(cfg *ini.File) {

    for true {

        // connect to database if ontvanger->dbsav is defined
        isConnected := false
        if cfg.Section("ontvanger").HasKey("dbsav") &&
            cfg.Section("database").Key("type").String() == "mysql" {

            // Build connection string
            connStr := cfg.Section("database").Key("user").String() + ":" + cfg.Section("database").Key("pass").String() +
                "@/" + cfg.Section("database").Key("db").String() + "?charset=utf8"

            // Create connection pool
            db, err := sql.Open("mysql", connStr)
            if err != nil {
                log.Println(err)
            } else {
                isConnected = true
            }

            stmt, err = db.Prepare("INSERT INTO minew(dateTime, boxID, tagID, rssi, data) values(?,?,?,?,?)")
            if err != nil {
                log.Println(err)
                isConnected = false
            } else {
                isConnected = true
            }

            if isConnected {
                log.Println("Connected to database", cfg.Section("database").Key("db").String())
            } else {
                log.Println("Connection to database failed")
            }
        }

        for r := range c {

            if isConnected {
                _, err := stmt.Exec(r.dateTime, r.boxID, r.tagID, r.rssi, string(r.data))
                if err != nil {

                    // on error, leave the loop and try new database connection
                    log.Println("Insert failed", err)
                    break
                }
            }
            log.Printf("%+v\n", r)
        }
    }
}

func handleConn(conn net.Conn) {

    defer conn.Close()

    // Make a buffer to hold incoming data.
    var p packet
    p.content = make([]byte, 16384)
    var err error

    // Read the incoming connection into the buffer.
    p.remAddr = conn.RemoteAddr().String()
    p.reqLen, err = conn.Read(p.content)
    if err != nil {
        log.Println(p.remAddr, "closed connection:", err.Error())
        return
    }
    // log.Println("From", remAddr, ":", buf[:reqLen])

    // Analyze packet
    p.analyzeBLE()

    // Send a response back
    _, err = conn.Write([]byte("Message received: " + string(p.content)))
    if err != nil {
        log.Println("Error wrinting to", p.remAddr, ":", err.Error())
    }
}

func (p packet) analyzeBLE() {

    if p.content[0] == 187 && p.content[p.reqLen-1] == 221 && p.reqLen > 22 {

        p.datal = binary.BigEndian.Uint32(p.content[2:6])
        p.boxID = hex.EncodeToString(p.content[6:12])
        p.nrBLE = binary.BigEndian.Uint16(p.content[12:14])
    } else {
        log.Println("Ignoring corrupted packet")
        return
    }

    ble := p.content[14:]

    for i := p.nrBLE; i > 0; i-- {

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
            db.rssi = int8(ble[rawLen+2])

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

func sqltime(ts int64) string {

    // return timestamp formatted for mssql or mysql
    const sqlts = "2006-01-02 15:04:05"
    loc, _ := time.LoadLocation("UTC")
    t := time.Unix(ts, 0).In(loc)
    return t.Format(sqlts)
}
