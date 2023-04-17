package main

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ip2location/ip2location-go/v9"
)

const (
	URL = "https://download.ip2location.com/lite/IP2LOCATION-LITE-DB1.IPV6.CSV.ZIP"

	workPath = "../testdata"
	csvName  = "IP2LOCATION-LITE-DB1.IPV6.CSV"
	zipName  = csvName + ".ZIP"
)

func downloadDB() error {
	zipPath := filepath.Join(workPath, zipName)
	f, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}
	resp, err := http.Get(URL)
	if err != nil {
		f.Close()
		os.Remove(zipPath)
		return err
	}
	defer resp.Body.Close()
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractDB() (err error) {
	var z *zip.ReadCloser
	for i := 0; i < 1; i++ {
		z, err = zip.OpenReader(filepath.Join(workPath, zipName))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = downloadDB()
				if err != nil {
					return
				}
				continue
			}
			return
		}
		break
	}
	if z == nil {
		return
	}

	defer z.Close()
	var csvFile io.ReadCloser
	for _, f := range z.File {
		if f.Name != "IP2LOCATION-LITE-DB1.IPV6.CSV" {
			continue
		}
		csvFile, err = f.Open()
		if err != nil {
			return err
		}
		break
	}

	if csvFile == nil {
		return errors.New("no CSV file found in the archive")
	}
	defer csvFile.Close()
	out, err := os.Create(filepath.Join(workPath, csvName))
	if err != nil {
		return
	}
	_, err = io.Copy(out, csvFile)
	return
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Missing argument")
	}
	country := os.Args[1]
	fileName4 := filepath.Join(workPath, fmt.Sprintf("%s_ipv4.txt", country))
	fileName6 := filepath.Join(workPath, fmt.Sprintf("%s_ipv6.txt", country))
	_, err := os.Stat(fileName4)
	if err == nil {
		_, err = os.Stat(fileName6)
		if err == nil {
			os.Exit(0)
		}
	}
	csvPath := filepath.Join(workPath, csvName)
	_, err = os.Stat(csvPath)
	if errors.Is(err, os.ErrNotExist) {
		err = extractDB()
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Open(csvPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	file4, err := os.Create(fileName4)
	if err != nil {
		log.Fatal(err)
	}
	defer file4.Close()

	file4w := bufio.NewWriter(file4)

	file6, err := os.Create(fileName6)
	if err != nil {
		log.Fatal(err)
	}
	defer file6.Close()

	file6w := bufio.NewWriter(file6)

	r := csv.NewReader(f)
	t := ip2location.OpenTools()

	var start, end big.Int
	line := 0
	for {
		rec, err := r.Read()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}
		line++
		if rec[2] != country {
			continue
		}

		if _, ok := start.SetString(rec[0], 10); !ok {
			log.Fatalf("At %d: Invalid int: %s", line, rec[0])
		}
		startIP, err := t.DecimalToIPv6(&start)
		if err != nil {
			log.Fatalf("At %d: %v", line, err)
		}
		if _, ok := end.SetString(rec[1], 10); !ok {
			log.Fatalf("At %d: Invalid int: %s", line, rec[1])
		}
		endIP, err := t.DecimalToIPv6(&end)
		if err != nil {
			log.Fatalf("At %d: %v", line, err)
		}

		var prefixes []string
		var w *bufio.Writer

		if t.IsIPv4(startIP) {
			prefixes, err = t.IPv4ToCIDR(startIP, endIP)
			w = file4w
		} else {
			prefixes, err = t.IPv6ToCIDR(startIP, endIP)
			w = file6w
		}
		if err != nil {
			log.Fatalf("At %d (%s-%s): %v", line, startIP, endIP, err)
		}

		for _, prefix := range prefixes {
			_, err = w.WriteString(prefix)
			if err != nil {
				log.Fatalf("Writing has failed: %v", err)
			}
			err = w.WriteByte('\n')
			if err != nil {
				log.Fatalf("Writing has failed: %v", err)
			}
		}
	}
	err = file4w.Flush()
	if err != nil {
		log.Fatal(err)
	}

	err = file6w.Flush()
	if err != nil {
		log.Fatal(err)
	}
}
