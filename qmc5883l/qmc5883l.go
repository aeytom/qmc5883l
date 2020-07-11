package qmc5883l

// qmc58883l implements http://wiki.sunfounder.cc/images/7/72/QMC5883L-Datasheet-1.0.pdf

import (
	"errors"
	"log"

	"github.com/d2r2/go-i2c"
	"github.com/d2r2/go-logger"
)

// default i2c bus number and address
const (
	DfltBus     = 1
	DfltAddress = 0x0d
)

// Output Data Registers for magnetic sensor.
const (
	RegXoutLSB = 0x00 // x-axis LSB
	RegYoutLSB = 0x02 // y-axis LSB
	RegZoutLSB = 0x04 // z-axis LSB
)

// Output Data Registers for temperature.
const (
	RegToutLSB = 0x07 // temperature LSB
)

// Status Register #1.
const (
	RegStatus1 = 0x06 // Status Register.
	StatDRDY   = 0x01 // Data Ready.
	StatOVL    = 0x02 // Overflow flag.
	StatDOR    = 0x04 // Data skipped for reading.
)

// Control Register 1
const (
	RegControl1 = 0x09 // Control Register #1.
	ModeSTBY    = 0x00 // Standby mode.
	ModeCONT    = 0x01 // Continuous read mode.
	Odr10HZ     = 0x00 // Output Data Rate Hz.
	Odr50HZ     = 0x04
	Odr100HZ    = 0x08
	Odr200HZ    = 0x0c
	Rng2G       = 0x00 // Range 2 Gauss: for magnetic-clean environments.
	Rng8G       = 0x10 // Range 8 Gauss: for strong magnetic fields.
	Osr512      = 0x00 // Over Sample Rate 512: less noise, more power.
	Osr256      = 0x40
	Osr128      = 0x80
	Osr64       = 0xc0 // Over Sample Rate 64: more noise, less power.
)

// Control Register 2
const (
	RegControl2 = 0x0a // Control Register #2.
	IntEnb      = 0x01 // Interrupt Pin Enabling.
	PolPnt      = 0x40 // Pointer Roll-over.
	SoftRst     = 0x80 // Soft Reset.
)

// Control Registers
const (
	RegRstPeriod = 0x0b // SET/RESET Period Register.
	RegChipID    = 0x0d // Chip ID register.
)

// QMC5883L chip handle
type QMC5883L struct {
	i2cBus           int
	address          byte
	outputDataRate   byte
	outputRange      byte
	oversamplingRate byte
	bus              *i2c.I2C
}

// New initilize structure
func New(i2cBus int, address uint8) *QMC5883L {
	if i2cBus == 0 {
		i2cBus = DfltBus
	}
	if address == 0 {
		address = DfltAddress
	}
	// Uncomment/comment next line to suppress/increase verbosity of output
	logger.ChangePackageLogLevel("i2c", logger.InfoLevel)

	bus, err := i2c.NewI2C(address, i2cBus)
	if err != nil {
		log.Fatal(err)
	}

	q := QMC5883L{
		i2cBus:           i2cBus,
		address:          address,
		outputDataRate:   Odr10HZ,
		outputRange:      Rng2G,
		oversamplingRate: Osr512,
		bus:              bus,
	}

	err = bus.WriteRegU8(RegRstPeriod, 0x01)
	if err != nil {
		log.Fatal(err)
	}

	err = bus.WriteRegU8(RegControl2, SoftRst)
	if err != nil {
		log.Fatal(err)
	}

	q.SetMode(ModeCONT, q.outputDataRate, q.outputRange, q.oversamplingRate)
	return &q
}

// Close the I²C handle
func (q *QMC5883L) Close() error {
	err := q.bus.Close()
	q.bus = nil
	return err
}

// SetMode set operating mode
func (q *QMC5883L) SetMode(mode byte, odr byte, rng byte, osr byte) error {
	q.outputDataRate = odr
	q.outputRange = rng
	q.oversamplingRate = osr
	err := q.bus.WriteRegU8(RegControl1, mode|odr|rng|osr)
	return err
}

// ReadRegistry read a byte value
func (q *QMC5883L) ReadRegistry(registry byte) (byte, error) {
	b, err := q.bus.ReadRegU8(registry)
	return b, err
}

// ReadWord read a two bytes value stored as LSB and MSB.
func (q *QMC5883L) ReadWord(registry byte) (int16, error) {
	val, err := q.bus.ReadRegU16LE(registry)
	return complement2(val), err
}

// GetMagnetRaw Get the 3 axis values from magnetic sensor.
func (q *QMC5883L) GetMagnetRaw() (int16, int16, int16, error) {
	var (
		x   int16
		y   int16
		z   int16
		err error
	)
	status, err := q.ReadRegistry(RegStatus1)
	if err == nil {
		if status&StatOVL == StatOVL {
			msg := "Magnetic sensor overflow."
			if q.outputRange == Rng2G {
				msg += " Consider switching to Rng8G output range."
			}
			err = errors.New(msg)
		} else if status&StatDRDY == StatDRDY {
			x, err = q.ReadWord(RegXoutLSB)
			if err != nil {
				return x, y, z, err
			}
			y, err = q.ReadWord(RegYoutLSB)
			if err != nil {
				return x, y, z, err
			}
			z, err = q.ReadWord(RegZoutLSB)
		} else if status&StatDOR == StatDOR {
			_, err = q.ReadWord(RegToutLSB)
			if err == nil {
				err = errors.New("data skipped for reading")
			}
		}
	}
	return x, y, z, err
}

// Complement2 Calculate the 2's complement of a two bytes value.
func complement2(val uint16) int16 {
	if val >= 0x8000 {
		return int16(int(val) - 0x10000)
	}
	return int16(val)
}
