//
// The MIT License (MIT)
//
// Copyright (c) 2017 Michael Tratz <support@esosoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package main

import (
	"flag"
	"github.com/jinzhu/configor"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
)

const Version = "1.2.0"

var debug = flag.Bool("debug", false, "enable debug logging")

type SQLQueries struct {
    Sql	string
}

type User struct {
	username string
	domain string
	ip string
}

var Config = struct {

	DB struct {
		Host	 string
		Port	 uint16 `default:"3306"`
		Socket	 string
		Name	 string
		User	 string `default:"root"`
		Password string `required:"true"`
		Options  string `default:"timeout=5s&collation=utf8mb4_unicode_ci"`
		Queries map[string]SQLQueries
		ConnectionMaxLifeTime	string	`default:"0"`
		MaxIdleConnections	int	`default:"2"`
		MaxOpenConnections	int	`default:"5"`
	}
}{}

func checklength(str string) bool {
        if len(strings.TrimSpace(str)) == 0 {
		return false
        }
	return true
}


func main() {
	config := flag.String("config", "/etc/postlogin.toml", "path to configuration file")
	flag.Parse()

	_, err := os.Stat(*config)
	if err != nil {
		log.Fatal("Config file does not exist ", *config)
	}

	userenv, _ := os.LookupEnv("USER")
	if !checklength(userenv) {
		log.Fatal("Username is empty ", userenv)
	}

	if !strings.Contains(userenv, "@") {
		log.Fatal("Username is invalid must be valid email address ", userenv)
	}

	user := new(User)
	s := strings.Split(userenv, "@")
	user.username = s[0]
	user.domain = s[1]
	
	if !checklength(user.username) || !checklength(user.domain) {
		log.Fatal("Username is invalid must be valid email address ", userenv)
	}

	ip, _ := os.LookupEnv("IP")
	valid_ip := net.ParseIP(ip)
	if valid_ip == nil {
		log.Fatal("Invalid IP Address ", ip)
	}
	user.ip = ip

	configor.Load(&Config, *config)

	db, err := connectDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.conn.Close()

	for _, sql := range db.queries {
		defer sql.Close()
        }

	err = db.addIP(user)
	if err != nil {
		if *debug {
                	log.Println("Unable to add IP address ", err)
        	}
		os.Exit(1)
	}

	env := os.Environ()
	args := os.Args
	binary := args[len(args)-1]

	err = syscall.Exec(binary, nil, env)
	if err != nil {
		if *debug {
                        log.Println("Unable to execute binary ", binary, err)
                }
		os.Exit(1)
	}
}
