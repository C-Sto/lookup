package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

type config struct {
	InputList string
	Localpath string
	IPs       bool
	No6       bool
	Threads   int
	Wg        *sync.WaitGroup
}

var conf config

func main() {

	flag.StringVar(&conf.InputList, "iL", "", "File to use as an input list of domains to resolve")
	flag.StringVar(&conf.Localpath, "o", "."+string(os.PathSeparator)+"resolved.txt", "Local file to dump successful resolves into (v6 resolves will go into a file with 6_ prepended)")
	flag.BoolVar(&conf.No6, "no6", false, "Don't do v6 resolves") //future :sunglasses:
	flag.BoolVar(&conf.IPs, "ip", false, "Append resolved IP to looked up domain")
	flag.IntVar(&conf.Threads, "t", 100, "Number of looker upper workers to use")
	flag.Parse()

	conf.Wg = &sync.WaitGroup{}

	//get words from local file
	file, err := os.Open(conf.InputList)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	filebytes, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(filebytes), "\n")
	lookupChan := make(chan string, 1000)
	writeChan := make(chan string, 1000)

	go writerDowner(writeChan, conf.Localpath)
	for x := 0; x < conf.Threads; x++ {
		go lookerUpper(lookupChan, writeChan)
	}

	for _, line := range lines {
		conf.Wg.Add(1)
		lookupChan <- line
	}
	conf.Wg.Wait()

}

func lookerUpper(lookupChan, writeChan chan string) {
	for {
		select {
		case domain := <-lookupChan:
			domain = strings.TrimSpace(domain)
			if len(domain) > 1 {

				//clean domains that start with a dot for some reason?
				if string(domain[0]) == "." {
					domain = domain[1:]
				}
				ips, err := net.LookupIP(domain)
				if err == nil && len(ips) > 0 {
					conf.Wg.Add(1)
					if conf.IPs {
						writeChan <- domain + ":" + ips[0].String()
					} else {
						writeChan <- domain
					}
				}
			}
		}
		conf.Wg.Done()
	}
}

func writerDowner(writeChan chan string, path string) {
	for {
		select {
		case word := <-writeChan:
			fmt.Println(word)
			writeDomain(word, path)
		}
		conf.Wg.Done()
	}
}

func writeDomain(domain, fileName string) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic("Can't open file for reading, is something wrong?\n" + err.Error())
	}
	defer file.Close()

	file.WriteString(domain + "\n")
	file.Sync()
}
