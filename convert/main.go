package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

var sizes = make(map[uint32]uint32, 32)

func init() {
	var n uint32 = 0xFFFF_FFFF
	for i := uint32(0); i < 32; i++ {
		sizes[n] = i
		n >>= 1
	}
}

func main() {
	r := csv.NewReader(os.Stdin)
	for {
		rec, err := r.Read()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}
		if rec[2] != "US" {
			continue
		}
		start, err := strconv.ParseUint(rec[0], 10, 32)
		if err != nil {
			panic(err)
		}
		end, err := strconv.ParseUint(rec[1], 10, 32)
		if err != nil {
			panic(err)
		}
		size := uint32(end - start)
		if pl := sizes[size]; pl != 0 {
			fmt.Printf("%s/%d\n", numToIP(uint32(start)), pl)
		}
	}
}

func numToIP(n uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		n>>24,
		n>>16&0xFF,
		n>>8&0xFF,
		n&0xFF,
	)
}
