package rgbmatrix

/*
#cgo CFLAGS: -std=c99 -I${SRCDIR}/lib/rpi-rgb-led-matrix/include -DSHOW_REFRESH_RATE
#cgo LDFLAGS: -lrgbmatrix -L${SRCDIR}/lib/rpi-rgb-led-matrix/lib -lstdc++ -lm
#include <led-matrix-c.h>

void led_matrix_swap(struct RGBLedMatrix *matrix, struct LedCanvas *offscreen_canvas,
                     int width, int height, const uint32_t pixels[]) {


  int i, x, y;
  uint32_t color;
  for (x = 0; x < width; ++x) {
    for (y = 0; y < height; ++y) {
      i = x + (y * width);
      color = pixels[i];

      led_canvas_set_pixel(offscreen_canvas, x, y,
        (color >> 16) & 255, (color >> 8) & 255, color & 255);
    }
  }

  offscreen_canvas = led_matrix_swap_on_vsync(matrix, offscreen_canvas);
}

void set_show_refresh_rate(struct RGBLedMatrixOptions *o, int show_refresh_rate) {
  o->show_refresh_rate = show_refresh_rate != 0 ? 1 : 0;
}

void set_disable_hardware_pulsing(struct RGBLedMatrixOptions *o, int disable_hardware_pulsing) {
  o->disable_hardware_pulsing = disable_hardware_pulsing != 0 ? 1 : 0;
}

void set_inverse_colors(struct RGBLedMatrixOptions *o, int inverse_colors) {
  o->inverse_colors = inverse_colors != 0 ? 1 : 0;
}

*/
import "C"
import (
	"fmt"
	"image/color"
	"os"
	"unsafe"

	"github.com/robbydyer/rgbmatrix-rpi/emulator"
)

// DefaultConfig default WS281x configuration
var DefaultConfig = HardwareConfig{
	Rows:              32,
	Cols:              32,
	ChainLength:       1,
	Parallel:          1,
	PWMBits:           11,
	PWMLSBNanoseconds: 130,
	Brightness:        100,
	ScanMode:          Progressive,
}

// DefaultRuntimeOptions default WS281x runtime options
var DefaultRuntimeOptions = RuntimeOptions{
	GPIOSlowdown:   0,
	Daemon:         0,
	DropPrivileges: 1,
	DoGPIOInit:     true,
}

// HardwareConfig rgb-led-matrix configuration
type HardwareConfig struct {
	// Rows the number of rows supported by the display, so 32 or 16.
	Rows int
	// Cols the number of columns supported by the display, so 32 or 64 .
	Cols int
	// ChainLengthis the number of displays daisy-chained together
	// (output of one connected to input of next).
	ChainLength int
	// Parallel is the number of parallel chains connected to the Pi; in old Pis
	// with 26 GPIO pins, that is 1, in newer Pis with 40 interfaces pins, that
	// can also be 2 or 3. The effective number of pixels in vertical direction is
	// then thus rows * parallel.
	Parallel int
	// Set PWM bits used for output. Default is 11, but if you only deal with
	// limited comic-colors, 1 might be sufficient. Lower require less CPU and
	// increases refresh-rate.
	PWMBits int
	// Change the base time-unit for the on-time in the lowest significant bit in
	// nanoseconds.  Higher numbers provide better quality (more accurate color,
	// less ghosting), but have a negative impact on the frame rate.
	PWMLSBNanoseconds int // the DMA channel to use
	// Brightness is the initial brightness of the panel in percent. Valid range
	// is 1..100
	Brightness int
	// ScanMode progressive or interlaced
	ScanMode ScanMode // strip color layout
	// Disable the PWM hardware subsystem to create pulses. Typically, you don't
	// want to disable hardware pulsing, this is mostly for debugging and figuring
	// out if there is interference with the sound system.
	// This won't do anything if output enable is not connected to GPIO 18 in
	// non-standard wirings.
	DisableHardwarePulsing bool

	ShowRefreshRate bool
	InverseColors   bool

	// Name of GPIO mapping used
	HardwareMapping string

	// Limit refresh rate of LED panel. This will help on a loaded system
	// to keep a constant refresh rate. <= 0 for no limit.
	LimitRefreshRateHz int
}

func (c *HardwareConfig) geometry() (width, height int) {
	return c.Cols * c.ChainLength, c.Rows * c.Parallel
}

func (c *HardwareConfig) toC() *C.struct_RGBLedMatrixOptions {
	o := &C.struct_RGBLedMatrixOptions{}
	o.rows = C.int(c.Rows)
	o.cols = C.int(c.Cols)
	o.chain_length = C.int(c.ChainLength)
	o.parallel = C.int(c.Parallel)
	o.pwm_bits = C.int(c.PWMBits)
	o.pwm_lsb_nanoseconds = C.int(c.PWMLSBNanoseconds)
	o.brightness = C.int(c.Brightness)
	o.scan_mode = C.int(c.ScanMode)
	o.hardware_mapping = C.CString(c.HardwareMapping)
	o.limit_refresh_rate_hz = C.int(c.LimitRefreshRateHz)

	if c.ShowRefreshRate == true {
		C.set_show_refresh_rate(o, C.int(1))
	} else {
		C.set_show_refresh_rate(o, C.int(0))
	}

	if c.DisableHardwarePulsing == true {
		C.set_disable_hardware_pulsing(o, C.int(1))
	} else {
		C.set_disable_hardware_pulsing(o, C.int(0))
	}

	if c.InverseColors == true {
		C.set_inverse_colors(o, C.int(1))
	} else {
		C.set_inverse_colors(o, C.int(0))
	}

	return o
}

type ScanMode int8

const (
	Progressive ScanMode = 0
	Interlaced  ScanMode = 1
)

type RuntimeOptions struct {
	// 0 = no slowdown. (Available 0...4)
	GPIOSlowdown int

	// ----------
	// If the following options are set to disabled with -1, they are not
	// even offered via the command line flags.
	// ----------

	// Thre are three possible values here
	//   -1 : don't leave choise of becoming daemon to the command line parsing.
	//        If set to -1, the --led-daemon option is not offered.
	//    0 : do not becoma a daemon, run in forgreound (default value)
	//    1 : become a daemon, run in background.
	//
	// If daemon is disabled (= -1), the user has to call
	// RGBMatrix::StartRefresh() manually once the matrix is created, to leave
	// the decision to become a daemon
	// after the call (which requires that no threads have been started yet).
	// In the other cases (off or on), the choice is already made, so the thread
	// is conveniently already started for you.
	// -1 disabled. 0=off, 1=on.
	Daemon int

	// Drop privileges from 'root' to 'daemon' once the hardware is initialized.
	// This is usually a good idea unless you need to stay on elevated privs.
	DropPrivileges int

	// By default, the gpio is initialized for you, but if you run on a platform
	// not the Raspberry Pi, this will fail. If you don't need to access GPIO
	// e.g. you want to just create a stream output (see content-streamer.h),
	// set this to false.
	DoGPIOInit bool
}

func (rto *RuntimeOptions) toC() *C.struct_RGBLedRuntimeOptions {
	o := &C.struct_RGBLedRuntimeOptions{}
	o.gpio_slowdown = C.int(rto.GPIOSlowdown)
	o.daemon = C.int(rto.Daemon)
	o.drop_privileges = C.int(rto.DropPrivileges)
	o.do_gpio_init = C.bool(rto.DoGPIOInit)

	return o
}

// RGBLedMatrix matrix representation for ws281x
type RGBLedMatrix struct {
	Config         *HardwareConfig
	RuntimeOptions *RuntimeOptions

	height int
	width  int
	matrix *C.struct_RGBLedMatrix
	buffer *C.struct_LedCanvas
	leds   []C.uint32_t
}

const MatrixEmulatorENV = "MATRIX_EMULATOR"

// NewRGBLedMatrix returns a new matrix using the given size and config
func NewRGBLedMatrix(config *HardwareConfig, rtOptions *RuntimeOptions) (c Matrix, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("error creating matrix: %v", r)
			}
		}
	}()

	if isMatrixEmulator() {
		return buildMatrixEmulator(config), nil
	}

	w, h := config.geometry()
	m := C.led_matrix_create_from_options_and_rt_options(config.toC(), rtOptions.toC())
	b := C.led_matrix_create_offscreen_canvas(m)
	c = &RGBLedMatrix{
		Config: config,
		width:  w, height: h,
		matrix: m,
		buffer: b,
		leds:   make([]C.uint32_t, w*h),
	}
	if m == nil {
		return nil, fmt.Errorf("unable to allocate memory")
	}

	return c, nil
}

func isMatrixEmulator() bool {
	if os.Getenv(MatrixEmulatorENV) == "1" {
		return true
	}

	return false
}

func buildMatrixEmulator(config *HardwareConfig) Matrix {
	w, h := config.geometry()
	return emulator.NewEmulator(w, h, emulator.DefaultPixelPitch, true)
}

// Initialize initialize library, must be called once before other functions are
// called.
func (c *RGBLedMatrix) Initialize() error {
	return nil
}

// Geometry returns the width and the height of the matrix
func (c *RGBLedMatrix) Geometry() (width, height int) {
	return c.width, c.height
}

// Apply set all the pixels to the values contained in leds
func (c *RGBLedMatrix) Apply(leds []color.Color) error {
	for position, l := range leds {
		c.Set(position, l)
	}

	return c.Render()
}

// Render update the display with the data from the LED buffer
func (c *RGBLedMatrix) Render() error {
	w, h := c.Config.geometry()

	C.led_matrix_swap(
		c.matrix,
		c.buffer,
		C.int(w), C.int(h),
		(*C.uint32_t)(unsafe.Pointer(&c.leds[0])),
	)

	c.leds = make([]C.uint32_t, w*h)
	return nil
}

// At return an Color which allows access to the LED display data as
// if it were a sequence of 24-bit RGB values.
func (c *RGBLedMatrix) At(position int) color.Color {
	return uint32ToColor(c.leds[position])
}

// Set set LED at position x,y to the provided 24-bit color value.
func (c *RGBLedMatrix) Set(position int, color color.Color) {
	c.leds[position] = C.uint32_t(colorToUint32(color))
}

// Close finalizes the ws281x interface
func (c *RGBLedMatrix) Close() error {
	C.led_matrix_delete(c.matrix)
	return nil
}

func (c *RGBLedMatrix) SetBrightness(brightness int) {
	C.led_matrix_set_brightness(c.matrix, C.uint8_t(brightness))
}

func colorToUint32(c color.Color) uint32 {
	if c == nil {
		return 0
	}

	// A color's RGBA method returns values in the range [0, 65535]
	red, green, blue, _ := c.RGBA()
	return (red>>8)<<16 | (green>>8)<<8 | blue>>8
}

func uint32ToColor(u C.uint32_t) color.Color {
	return color.RGBA{
		uint8(u>>16) & 255,
		uint8(u>>8) & 255,
		uint8(u>>0) & 255,
		0,
	}
}
