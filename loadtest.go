package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var TARGETS_FILE = "./targets"

var DURATION = "20s"
var RATE = "500"

func copyVegeta(host string) {
	exec.Command("scp", TARGETS_FILE, host+":/tmp/.").Run()
	exec.Command("scp", "./vegeta", host+":/tmp/.").Run()
}

func runLoadtest(host string, outFile io.Writer) error {
	c := exec.Command("ssh", host,
		"/tmp/vegeta attack"+
			" -duration="+DURATION+
			" -targets /tmp/targets"+
			" -rate="+RATE,
	)
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(outFile, stdout); err != nil {
		return err
	}

	if err := c.Wait(); err != nil {
		return err
	}

	return nil
}

func printReport(files []string) {
	o, _ := exec.Command("./vegeta", "report", "-input", strings.Join(files, ",")).CombinedOutput()
	fmt.Print(string(o))
}

func removeTmpFiles(files []string) {
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			log.Print(err)
		}
	}
}

func mapHosts(f func(string) string, hosts []string) []string {
	resChan := make(chan string)
	for _, h := range hosts {
		go func(h string) {
			resChan <- f(h)
		}(h)
	}

	res := make([]string, 0)
	for _, _ = range hosts {
		tmp := <-resChan
		if tmp != "" {
			res = append(res, tmp)
		}
	}

	return res
}

func main() {
	hosts := os.Args[1:]

	log.Println("Copying vegeta...")
	mapHosts(func(h string) string {
		copyVegeta(h)
		return ""
	}, hosts)

	log.Println("Running loadtest for " + DURATION + "...")
	files := mapHosts(func(h string) string {
		outFile, err := ioutil.TempFile("", "loadtest")
		if err != nil {
			return ""
		}
		defer outFile.Close()

		err = runLoadtest(h, outFile)
		if err != nil {
			log.Print(err)
			os.Remove(outFile.Name())
			return ""
		}
		return outFile.Name()
	}, hosts)

	printReport(files)

	removeTmpFiles(files)
}
