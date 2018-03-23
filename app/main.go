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
	"fmt"
	"time"

	"./menu"

	"github.com/the-sibyl/goLCD20x4"
	"github.com/the-sibyl/softStepper"
	"github.com/the-sibyl/sysfsGPIO"
)

func main() {
	var pg PlateGenie

	// Set up the display
	lcd := goLCD20x4.Open(2, 3, 4, 17, 27, 22, 10, 9, 11, 0, 5)
	defer lcd.Close()

	lcd.FunctionSet(1, 1, 0)
	lcd.DisplayOnOffControl(1, 0, 0)
	lcd.EntryModeSet(1, 0)

	lcd.ClearDisplay()

	lcd.WriteLine("Welcome to", 1)
	lcd.WriteLine("PLATE GENIE", 2)

	m := menu.CreateMenu(lcd)
	m.AddMenuItem("Home Both", "", "", "   GO  ", "  GO   ")
	m.AddMenuItem("Home Single", "", "", " Left  ", " Right ")
	m.AddMenuItem("Move to Center", "", "", "   GO  ", "  GO   ")
	m.AddMenuItem("Speed", "(% Max Speed)", "100%", "   INC ", " DEC   ")
	m.AddMenuItem("Travel", "(% Max Distance)", "100%", "   INC ", " DEC   ")
/*
	go func() {
		for {
		m.Next()
		time.Sleep(time.Second * 1)
		}

	} ()
*/

	// GPIO 6, 13, 19, 26 for membrane keypad
	gpio6, _ := sysfsGPIO.InitPin(6, "in")
	defer gpio6.ReleasePin()
	gpio6.SetTriggerEdge("rising")
	gpio6.AddPinInterrupt()

	gpio13, _ := sysfsGPIO.InitPin(13, "in")
	defer gpio13.ReleasePin()
	gpio13.SetTriggerEdge("rising")
	gpio13.AddPinInterrupt()

	gpio19, _ := sysfsGPIO.InitPin(19, "in")
	defer gpio19.ReleasePin()
	gpio19.SetTriggerEdge("rising")
	gpio19.AddPinInterrupt()

	gpio26, _ := sysfsGPIO.InitPin(26, "in")
	defer gpio26.ReleasePin()
	gpio26.SetTriggerEdge("rising")
	gpio26.AddPinInterrupt()

	// GPIO 18 for the green button and GPIO 23 for the red button
	gpio18, _ := sysfsGPIO.InitPin(18, "in")
	defer gpio18.ReleasePin()
	gpio18.SetTriggerEdge("rising")
	gpio18.AddPinInterrupt()

	gpio23, _ := sysfsGPIO.InitPin(23, "in")
	defer gpio23.ReleasePin()
	gpio23.SetTriggerEdge("rising")
	gpio23.AddPinInterrupt()


	// GPIO 21 for left limit switch. GPIO 16 for right limit switch.
	// Pull-ups are defined in the device tree overlay.
	gpioLeftLimit, _ := sysfsGPIO.InitPin(21, "in")
	defer gpioLeftLimit.ReleasePin()
	gpioLeftLimit.SetTriggerEdge("both")
	gpioLeftLimit.AddPinInterrupt()
	pg.gpioLeftLimit = gpioLeftLimit

	gpioRightLimit, _ := sysfsGPIO.InitPin(16, "in")
	defer gpioRightLimit.ReleasePin()
	gpioRightLimit.SetTriggerEdge("both")
	gpioRightLimit.AddPinInterrupt()
	pg.gpioRightLimit = gpioRightLimit

// TODO: Using an obscenely long amount of time delay doesn't fix this problem. Figure out a deterministic solution
// to discard the first few interrupts.
	// A trigger event will happen once everything is set up but before the user has actually pressed a button
	time.Sleep(time.Millisecond * 2000)
	<-sysfsGPIO.GetInterruptStream()

	go func() {
		for {
			select {
				case s := <-sysfsGPIO.GetInterruptStream():
					switch(s.IOPin.GPIONum) {
						// Button 1
						case 19:
							m.Prev()
//							lcd.WriteLine("Button 1 pressed last", 4)
						// Button 2
						case 26:
//							lcd.WriteLine("Button 2 pressed last", 4)
						// Button 3
						case 6:
//							lcd.WriteLine("Button 3 pressed last", 4)
						// Button 4
						case 13:
//							lcd.WriteLine("Button 4 pressed last", 4)
							m.Next()
						case 21:
							fmt.Println("Left limit hit")
						case 16:
							fmt.Println("Right limit hit")
						case 18:
							fmt.Println("Green button hit")
						case 23:
							fmt.Println("Red button hit")
					}
			}
		}
	} ()

//TODO: put stepperSpeed back in its proper place
const stepperSpeed = 1000
	stepper1 := softStepper.InitStepperTwoEnaPins(24, 12, 25, 8, 7, 1, time.Microsecond*stepperSpeed)
	defer stepper1.ReleaseStepper()

	stepper1.EnableHold()

	pg.stepper = stepper1

	pg.homeBoth()

	for {
		time.Sleep(time.Second)
	}

//	for {
//		moveTrapezoidal(stepper1, 60.105)
//		moveTrapezoidal(stepper1, -60.105)
//		/*
//			move(stepper1, -0.5)
//			move(stepper1, 0.5)
//		*/
//	}

}

