package main

import (
    "database/sql"
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
    rssi     int
    batt     int
    data     string
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
    var format string
    if cfg.Section("ontvanger").HasKey("format") {
        format = cfg.Section("ontvanger").Key("format").String()
    } else {
        log.Fatalln("Missing format in configuration (binary/json)")
    }

    var handle func(net.Conn, chan dbRecord)
    switch format {
    case "binary", "bin":
        handle = handleBINConn
    case "json":
        handle = handleJSONConn
    default:
        log.Fatalf("Unknown format %s, should be either json or binary\n")
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

        go handle(conn, c)
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

            stmt, err = db.Prepare("INSERT INTO minew(dateTime, boxID, tagID, rssi, battery, data) values(?,?,?,?,?,?)")
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
                _, err := stmt.Exec(r.dateTime, r.boxID, r.tagID, r.rssi, r.batt, string(r.data))
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

func sqltime(ts int64) string {

    // return timestamp formatted for mssql or mysql
    const sqlts = "2006-01-02 15:04:05"
    loc, _ := time.LoadLocation("UTC")
    t := time.Unix(ts, 0).In(loc)
    return t.Format(sqlts)
}
