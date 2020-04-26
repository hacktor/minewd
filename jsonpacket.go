package main

import (
    "encoding/json"
    //"encoding/hex"
    "log"
    "net"

    _ "github.com/go-sql-driver/mysql"
)

type packet struct {
    content []byte
    remAddr string
    reqLen  int
    boxID   string
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

    // p.content is not quite json yet, so this will fail
    var tags map[string]interface{}
    err := json.Unmarshal([]byte(p.content), &tags)
    if err != nil {
        log.Println("json error:", err.Error())
        return
    }
    log.Println(tags)
}

