package main

import (
	"fmt"
	fsnotify "github.com/fsnotify/fsnotify"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/oschwald/geoip2-golang"
	"github.com/satyrius/gonx"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const logsPath = "./"
const logPattern = "proxy-host-*_access.log"
const nginxConfig = "nginx/nginx.conf"

func main() {
	// create reader for /etc/nginx/nginx.conf
	db, _ := geoip2.Open("GeoLite2-City.mmdb")
	defer db.Close()

	file, err := os.Open(nginxConfig)
	if err != nil {
		log.Fatal("Could not read nginx config", err)
	}
	parser, err := gonx.NewNginxParser(file, "proxy")
	if err != nil {
		log.Fatal("Could not create nginx log parser: ", err)
	}

	bucket := os.Getenv("INFLUXDB_BUCKET")
	org := os.Getenv("INFLUXDB_ORG")
	token := os.Getenv("INFLUXDB_TOKEN")
	url := os.Getenv("INFLUXDB_URL")

	client := influxdb2.NewClient(url, token)

	writeApi := client.WriteAPI(org, bucket)

	listenForFileModifications(func(line string) error {
		entry, err := parser.ParseString(strings.TrimSpace(line))

		if err != nil {
			return nil
		}
		timeString, _ := entry.Field("time_local")

		time, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeString)
		if err != nil {
			log.Fatal("Could not parse time: ", err)
		}

		upstreamStatus, err := entry.Field("upstream_status")
		status, err := entry.Field("status")
		requestMethod, err := entry.Field("request_method")
		scheme, err := entry.Field("scheme")
		host, err := entry.Field("host")
		requestUri, err := entry.Field("request_uri")
		remoteAddr, err := entry.Field("remote_addr")
		bodyBytesSent, err := entry.Field("body_bytes_sent")
		gzipRation, err := entry.Field("gzip_ratio")
		server, err := entry.Field("server")
		httpUserAgent, err := entry.Field("http_user_agent")
		httpReferer, err := entry.Field("http_referer")

		ip := net.ParseIP(remoteAddr)

		cityName := ""
		stateName := ""
		country := ""
		latitude := 0.0
		longitude := 0.0
		postalCode := ""

		if !ip.IsPrivate() && !ip.IsLoopback() {
			record, _ := db.City(ip)
			cityName = record.City.Names["en"]
			stateName = record.Subdivisions[0].Names["en"]
			country = record.Country.Names["en"]
			latitude = record.Location.Latitude
			longitude = record.Location.Longitude
			postalCode = record.Postal.Code
		}

		fmt.Printf("IP %s, Target: %s, Country: %s, City: %s\n", ip, host, country, cityName)

		point := influxdb2.NewPoint("nginx_access_log", map[string]string{
			"target":      host,
			"client_ip":   remoteAddr,
			"city":        cityName,
			"state":       stateName,
			"country":     country,
			"latitude":    fmt.Sprintf("%f", latitude),
			"longitude":   fmt.Sprintf("%f", longitude),
			"postal_code": postalCode,
		}, map[string]interface{}{
			"upstream_status": upstreamStatus,
			"status":          status,
			"method":          requestMethod,
			"scheme":          scheme,
			"target":          host,
			"uri":             requestUri,
			"client_ip":       remoteAddr,
			"body_bytes":      bodyBytesSent,
			"gzip_ratio":      gzipRation,
			"server":          server,
			"user_agent":      httpUserAgent,
			"referer":         httpReferer,
			"request_time":    timeString,
			"city":            cityName,
			"state":           stateName,
			"country":         country,
			"latitude":        latitude,
			"longitude":       longitude,
			"postal_code":     postalCode,
		}, time)

		writeApi.WritePoint(point)
		return nil
	})

	// wait for the program to exit
	select {}
}

func listenForFileModifications(callback func(string) error) {
	// create a new file watcher, use fsnotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	done := make(chan bool)
	// watch the log file for new lines
	go func() {
		for {
			select {

			case event := <-watcher.Events:
				match, err := filepath.Match(logPattern, strings.Replace(event.Name, "./", "", 1))
				// if the event is a new line
				if event.Op&fsnotify.Write == fsnotify.Write && match {
					// read the new line
					line := readLine(event.Name)

					// call the callback function
					err = callback(line)
					if err != nil {
					}

				}
			case err := <-watcher.Errors:
				log.Fatal("error:", err)
			}
		}
	}()
	err = watcher.Add(logsPath)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func readLine(filepath string) string {
	fileHandle, err := os.Open(filepath)

	if err != nil {
		panic("Cannot open file")
		os.Exit(1)
	}
	defer fileHandle.Close()

	line := ""
	var cursor int64 = 0
	stat, _ := fileHandle.Stat()
	filesize := stat.Size()
	for {
		cursor -= 1
		fileHandle.Seek(cursor, io.SeekEnd)

		char := make([]byte, 1)
		fileHandle.Read(char)

		if cursor != -1 && (char[0] == 10 || char[0] == 13) { // stop if we find a line
			break
		}

		line = fmt.Sprintf("%s%s", string(char), line) // there is more efficient way

		if cursor == -filesize { // stop if we are at the begining
			break
		}
	}

	return line
}
func getLogFiles(path string, pattern string) []string {
	// create a slice to store the log files
	var logFiles []string
	// walk the logs directory
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		// if the file is a log file
		// pattern is a glob pattern, so we use filepath.Match to check if the file matches the pattern
		match, err := filepath.Match(pattern, info.Name())
		if !info.IsDir() && match {
			// add the file to the slice
			logFiles = append(logFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return logFiles
}
