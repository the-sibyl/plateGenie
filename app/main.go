/*
Copyright (c) 2018 Forrest Sibley <My^Name^Without^The^Surname@ieee.org>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"time"

	"github.com/the-sibyl/goLCD20x4"
	"github.com/the-sibyl/plateGenie"
	"github.com/the-sibyl/softStepper"
	"github.com/the-sibyl/sysfsGPIO"
)

func main() {
	// Set up the display
	lcd := goLCD20x4.Open(2, 3, 4, 17, 27, 22, 10, 9, 11, 0, 5)
	defer lcd.Close()

	// GPIO 6, 13, 19, 26 for membrane keypad
	gpio6, _ := sysfsGPIO.InitPin(6, "in")
	defer gpio6.ReleasePin()

	gpio13, _ := sysfsGPIO.InitPin(13, "in")
	defer gpio13.ReleasePin()

	gpio19, _ := sysfsGPIO.InitPin(19, "in")
	defer gpio19.ReleasePin()

	gpio26, _ := sysfsGPIO.InitPin(26, "in")
	defer gpio26.ReleasePin()

	// GPIO 18 for the green button and GPIO 23 for the red button
	gpio18, _ := sysfsGPIO.InitPin(18, "in")
	defer gpio18.ReleasePin()

	gpio23, _ := sysfsGPIO.InitPin(23, "in")
	defer gpio23.ReleasePin()


	// GPIO 21 for left limit switch. GPIO 16 for right limit switch.
	// Pull-ups are defined in the device tree overlay.
	gpio21, _ := sysfsGPIO.InitPin(21, "in")
	defer gpio21.ReleasePin()

	gpio16, _ := sysfsGPIO.InitPin(16, "in")
	defer gpio16.ReleasePin()

	// Placeholder -- TODO: come back and look at this
	const stepperSpeed = 10000
	stepper := softStepper.InitStepperTwoEnaPins(24, 12, 25, 8, 7, 1, time.Microsecond*stepperSpeed)

	plateGenie.Initialize(lcd, gpio19, gpio26, gpio6, gpio13, gpio18, gpio23, gpio21, gpio16, stepper)

	for {
		time.Sleep(time.Second)
	}
}
