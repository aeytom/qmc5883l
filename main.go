package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/aeytom/qmc5883l/qmc5883l"

	"github.com/stianeikeland/go-rpio"
)

var (
	// Verbose provide more debugging output
	Verbose *bool
	// Testing do not write influxdb
	Testing *bool
)

func main() {

	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()

	Verbose = flag.Bool("verbose", false, "provide more debugging output")
	Testing = flag.Bool("test", false, "do not write influxdb")

	flag.Parse()

	sensor := qmc5883l.New(qmc5883l.DfltBus, qmc5883l.DfltAddress)
	sensor.SetMode(qmc5883l.ModeCONT, qmc5883l.Odr200HZ, qmc5883l.Rng8G, qmc5883l.Osr512)

	for {
		x, y, z, err := sensor.GetMagnetRaw()
		log.Printf("x=%v y=%v z=%v err=%v", x, y, z, err)
		time.Sleep(time.Millisecond * 100)
	}
}

//
func getEnvArg(env string, arg string, dflt string, usage string) *string {
	ev, avail := os.LookupEnv(env)
	if avail {
		dflt = ev
	}
	v := flag.String(env, dflt, usage)
	return v
}
