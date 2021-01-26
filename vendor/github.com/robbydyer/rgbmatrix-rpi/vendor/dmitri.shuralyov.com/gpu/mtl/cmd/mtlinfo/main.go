// +build darwin

// mtlinfo is a tool that displays information about Metal devices in the system.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"dmitri.shuralyov.com/gpu/mtl"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: mtlinfo")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// Display the preferred system default Metal device.
	device, err := mtl.CreateSystemDefaultDevice()
	if err != nil {
		// An error here means Metal is not supported on this system.
		// Let the user know and stop here.
		fmt.Println(err)
		return nil
	}
	fmt.Println("preferred system default Metal device:", device.Name)

	// List all Metal devices in the system.
	allDevices := mtl.CopyAllDevices()
	for _, d := range allDevices {
		fmt.Println()
		printDeviceInfo(d)
	}

	return nil
}

func printDeviceInfo(d mtl.Device) {
	fmt.Println(d.Name + ":")
	fmt.Println("	• low-power:", yes(d.LowPower))
	fmt.Println("	• removable:", yes(d.Removable))
	fmt.Println("	• configured as headless:", yes(d.Headless))
	fmt.Println("	• registry ID:", d.RegistryID)
	fmt.Println()
	fmt.Println("	Feature Sets:")
	fmt.Println("	• macOS GPU family 1, version 1:", supported(d.SupportsFeatureSet(mtl.MacOSGPUFamily1V1)))
	fmt.Println("	• macOS GPU family 1, version 2:", supported(d.SupportsFeatureSet(mtl.MacOSGPUFamily1V2)))
	fmt.Println("	• macOS read-write texture, tier 2:", supported(d.SupportsFeatureSet(mtl.MacOSReadWriteTextureTier2)))
	fmt.Println("	• macOS GPU family 1, version 3:", supported(d.SupportsFeatureSet(mtl.MacOSGPUFamily1V3)))
	fmt.Println("	• macOS GPU family 1, version 4:", supported(d.SupportsFeatureSet(mtl.MacOSGPUFamily1V4)))
	fmt.Println("	• macOS GPU family 2, version 1:", supported(d.SupportsFeatureSet(mtl.MacOSGPUFamily2V1)))
}

func yes(v bool) string {
	switch v {
	case true:
		return "yes"
	case false:
		return "no"
	}
	panic("unreachable")
}

func supported(v bool) string {
	switch v {
	case true:
		return "✅ supported"
	case false:
		return "❌ unsupported"
	}
	panic("unreachable")
}
