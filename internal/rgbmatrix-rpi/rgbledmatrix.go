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
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/robbydyer/sports/internal/matrix"
	"go.uber.org/atomic"
	"go.uber.org/zap"
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
	Rows int `json:"rows"`
	// Cols the number of columns supported by the display, so 32 or 64 .
	Cols int `json:"cols"`
	// ChainLengthis the number of displays daisy-chained together
	// (output of one connected to input of next).
	ChainLength int `json:"chainLength"`
	// Parallel is the number of parallel chains connected to the Pi; in old Pis
	// with 26 GPIO pins, that is 1, in newer Pis with 40 interfaces pins, that
	// can also be 2 or 3. The effective number of pixels in vertical direction is
	// then thus rows * parallel.
	Parallel int `json:"parallel"`
	// Set PWM bits used for output. Default is 11, but if you only deal with
	// limited comic-colors, 1 might be sufficient. Lower require less CPU and
	// increases refresh-rate.
	PWMBits int `json:"pwmBits"`

	// The lower bits can be time-dithered for higher refresh rate.
	PWMDitherBits int `json:"pwmDitherBits"`

	// Change the base time-unit for the on-time in the lowest significant bit in
	// nanoseconds.  Higher numbers provide better quality (more accurate color,
	// less ghosting), but have a negative impact on the frame rate.
	PWMLSBNanoseconds int `json:"pwmlsbNanoseconds"` // the DMA channel to use
	// Brightness is the initial brightness of the panel in percent. Valid range
	// is 1..100
	Brightness int `json:"brightness"`
	// ScanMode progressive or interlaced
	ScanMode ScanMode `json:"scanMode"` // strip color layout
	// Disable the PWM hardware subsystem to create pulses. Typically, you don't
	// want to disable hardware pulsing, this is mostly for debugging and figuring
	// out if there is interference with the sound system.
	// This won't do anything if output enable is not connected to GPIO 18 in
	// non-standard wirings.
	DisableHardwarePulsing bool `json:"disableHardwarePulsing"`

	ShowRefreshRate bool   `json:"showRefreshRate"`
	InverseColors   bool   `json:"inverseColors"`
	LedRGBSequence  string `json:"ledRgbSequence"`

	// Name of GPIO mapping used
	HardwareMapping string `json:"hardwareMapping"`

	// Limit refresh rate of LED panel. This will help on a loaded system
	// to keep a constant refresh rate. <= 0 for no limit.
	LimitRefreshRateHz int `json:"limitRefreshRateHz"`

	// Type of multiplexing. 0 = direct, 1 = stripe, 2 = checker,...
	Multiplexing int `json:"multiplexing"`

	// A string describing a sequence of pixel mappers that should be applied
	// to this matrix. A semicolon-separated list of pixel-mappers with optional
	// parameter. See https://github.com/hzeller/rpi-rgb-led-matrix#panel-arrangement
	PixelMapperConfig string `json:"pixelMapperConfig"`

	// PanelType. Defaults to "". See https://github.com/hzeller/rpi-rgb-led-matrix#types-of-displays
	PanelType string `json:"panelType"`

	// RowAddr Type Adressing of rows; in particular panels with only AB address lines might indicate that this is needed.
	// See https://github.com/hzeller/rpi-rgb-led-matrix#types-of-displays for more info
	RowAddrType int `json:"rowAddrType"`
}

func (c *HardwareConfig) geometry() (width, height int) {
	col := 0
	row := 0
	if strings.EqualFold(c.PixelMapperConfig, "u-mapper") {
		if c.ChainLength > 1 {
			col = c.Cols * (c.ChainLength / 2)
			row = c.Rows * 2 * c.Parallel
		} else {
			col = c.Cols
			row = c.Rows * c.Parallel
		}
	} else if strings.EqualFold(c.PixelMapperConfig, "v-mapper") {
		if c.ChainLength > 1 {
			col = c.Cols * c.Parallel * 2
			row = c.Rows * (c.ChainLength / 2)
		} else {
			col = c.Cols * c.Parallel
			row = c.Rows
		}
	} else {
		col = c.Cols * c.ChainLength
		row = c.Rows * c.Parallel
	}

	return col, row
}

func (c *HardwareConfig) toC() *C.struct_RGBLedMatrixOptions {
	o := &C.struct_RGBLedMatrixOptions{}
	o.rows = C.int(c.Rows)
	o.cols = C.int(c.Cols)
	o.chain_length = C.int(c.ChainLength)
	o.parallel = C.int(c.Parallel)
	o.pwm_bits = C.int(c.PWMBits)
	o.pwm_dither_bits = C.int(c.PWMDitherBits)
	o.pwm_lsb_nanoseconds = C.int(c.PWMLSBNanoseconds)
	o.brightness = C.int(c.Brightness)
	o.scan_mode = C.int(c.ScanMode)
	o.hardware_mapping = C.CString(c.HardwareMapping)
	o.limit_refresh_rate_hz = C.int(c.LimitRefreshRateHz)
	o.row_address_type = C.int(c.RowAddrType)
	o.multiplexing = C.int(c.Multiplexing)

	if c.PanelType != "" {
		o.panel_type = C.CString(c.PanelType)
	}

	if c.PixelMapperConfig != "" {
		o.pixel_mapper_config = C.CString(c.PixelMapperConfig)
	}

	if c.LedRGBSequence != "" {
		o.led_rgb_sequence = C.CString(c.LedRGBSequence)
	}

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
	GPIOSlowdown int `json:"gpioSlowdown"`

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
	Daemon int `json:"daemon"`

	// Drop privileges from 'root' to 'daemon' once the hardware is initialized.
	// This is usually a good idea unless you need to stay on elevated privs.
	DropPrivileges int `json:"dropPrivileges"`

	// By default, the gpio is initialized for you, but if you run on a platform
	// not the Raspberry Pi, this will fail. If you don't need to access GPIO
	// e.g. you want to just create a stream output (see content-streamer.h),
	// set this to false.
	DoGPIOInit bool `json:"doGPIOInit"`
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

	height      int
	width       int
	matrix      *C.struct_RGBLedMatrix
	buffer      *C.struct_LedCanvas
	leds        []C.uint32_t
	preload     [][]C.uint32_t
	closed      *atomic.Bool
	log         *zap.Logger
	preloadLock sync.Mutex
	sync.Mutex
}

// NewRGBLedMatrix returns a new matrix using the given size and config
func NewRGBLedMatrix(config *HardwareConfig, rtOptions *RuntimeOptions, logger *zap.Logger) (c *RGBLedMatrix, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("error creating matrix: %v", r)
			}
		}
	}()

	w, h := config.geometry()
	m := C.led_matrix_create_from_options_and_rt_options(config.toC(), rtOptions.toC())
	b := C.led_matrix_create_offscreen_canvas(m)
	c = &RGBLedMatrix{
		Config: config,
		width:  w, height: h,
		matrix: m,
		buffer: b,
		leds:   make([]C.uint32_t, w*h),
		closed: atomic.NewBool(false),
		log:    logger,
	}
	if m == nil {
		return nil, fmt.Errorf("unable to allocate memory")
	}

	return c, nil
}

// Geometry returns the width and the height of the matrix
func (c *RGBLedMatrix) Geometry() (int, int) {
	return c.width, c.height
}

// Render update the display with the data from the LED buffer
func (c *RGBLedMatrix) Render() error {
	defer func() {
		w, h := c.Config.geometry()
		c.leds = make([]C.uint32_t, w*h)
	}()
	return c.render(c.leds)
}

func (c *RGBLedMatrix) render(leds []C.uint32_t) error {
	c.Lock()
	defer c.Unlock()

	if c.closed.Load() {
		return nil
	}

	// Check this so we don't cause a panic
	if len(leds) < 1 {
		return fmt.Errorf("led buffer is empty")
	}

	w, h := c.Config.geometry()

	C.led_matrix_swap(
		c.matrix,
		c.buffer,
		C.int(w),
		C.int(h),
		(*C.uint32_t)(unsafe.Pointer(&leds[0])),
	)

	return nil
}

// At return an Color which allows access to the LED display data as
// if it were a sequence of 24-bit RGB values.
func (c *RGBLedMatrix) At(x int, y int) color.Color {
	return uint32ToColor(c.leds[c.position(x, y)])
}

// Set set LED at position x,y to the provided 24-bit color value.
func (c *RGBLedMatrix) Set(x int, y int, color color.Color) {
	position := c.position(x, y)
	if position > len(c.leds)-1 || position < 0 {
		return
	}
	c.leds[position] = C.uint32_t(colorToUint32(color))
}

func (c *RGBLedMatrix) PreLoad(scene *matrix.MatrixScene) {
	c.preloadLock.Lock()
	defer c.preloadLock.Unlock()

	w, h := c.Config.geometry()
	prep := make([]C.uint32_t, w*h)

	for _, pt := range scene.Points {
		position := c.position(pt.X, pt.Y)
		prep[position] = C.uint32_t(colorToUint32(pt.Color))
	}

	if len(c.preload) < scene.Index+1 {
		newPreload := make([][]C.uint32_t, scene.Index+1)
		copy(newPreload, c.preload)
		c.preload = newPreload
	}

	c.preload[scene.Index] = prep
}

func (c *RGBLedMatrix) ReversePreLoad() {
	for i, j := 0, len(c.preload)-1; i < j; i, j = i+1, j-1 {
		c.preload[i], c.preload[j] = c.preload[j], c.preload[i]
	}
}

func (c *RGBLedMatrix) Play(ctx context.Context, startInterval time.Duration, interval <-chan time.Duration) error {
	defer func() {
		c.preload = [][]C.uint32_t{}
	}()
	waitInterval := startInterval
	c.log.Info("Play matrix",
		zap.Duration("default interval", waitInterval),
	)
	for _, leds := range c.preload {
		// An updated interval can be sent to the channel to change scroll speed
		select {
		case <-ctx.Done():
			return context.Canceled
		case waitInterval = <-interval:
			c.log.Info("RGB matrix got new interval during play",
				zap.Duration("interval", waitInterval),
			)
		default:
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(waitInterval):
		}

		if err := c.render(leds); err != nil {
			return err
		}
	}

	return nil
}

// Close finalizes the ws281x interface
func (c *RGBLedMatrix) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.closed.Load() {
		return nil
	}
	defer c.closed.Store(true)
	C.led_matrix_delete(c.matrix)
	return nil
}

func (c *RGBLedMatrix) SetBrightness(brightness int) {
	C.led_matrix_set_brightness(c.matrix, C.uint8_t(brightness))
}

func (c *RGBLedMatrix) position(x int, y int) int {
	return x + (y * c.width)
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

func uint32ToColorGo(u uint32) color.Color {
	return color.RGBA{
		uint8(u>>16) & 255,
		uint8(u>>8) & 255,
		uint8(u>>0) & 255,
		0,
	}
}
