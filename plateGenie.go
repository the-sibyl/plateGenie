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

package plateGenie

import (
	"fmt"
	"time"

	"github.com/the-sibyl/goLCD20x4"
	"github.com/the-sibyl/softStepper"
	"github.com/the-sibyl/sysfsGPIO"
)

const (
	// Stepper speed in microseconds
	stepperSpeed = 2000
	// Acceleration/deceleration window in microseconds for trapezoidal acceleration profile
	trapAccelPeriod = 50000
	// Minimum delay for the trapezoidal profile: this may be zero (fastest)
	trapMinimumDelay = 50
	// Maximum delay for the trapezoidal profile: this must not exceed the acceleration period
	trapMaximumDelay = 1000
	// The plater has a 500mm 4-start leadscrew with a 2mm pitch and 8mm lead.
	screwLead   = 8.0
	stepsPerRev = 200.0
	// The maximum acceptable distance for a function to accept. This is a sanity check value.
	maxDistance = 500.0
	// Maximum number of steps to be traversed for an axis move on a homing operation
	maxHomingSteps  = 100000
	// Number of steps to back-off in a homing operation
	backoffSteps = 50
	// Delay to slow down the stepper for homing movements in addition to the built-in delay from the stepper Initialize()
	homingStepDelay = 1000
)

type PlateGenie struct {
	lcd *goLCD20x4.LCD20x4

	gpioMembrane1 *sysfsGPIO.IOPin
	gpioMembrane2 *sysfsGPIO.IOPin
	gpioMembrane3 *sysfsGPIO.IOPin
	gpioMembrane4 *sysfsGPIO.IOPin

	gpioRedButton   *sysfsGPIO.IOPin
	gpioGreenButton *sysfsGPIO.IOPin

	gpioLeftLimit  *sysfsGPIO.IOPin
	gpioRightLimit *sysfsGPIO.IOPin

	stepper *softStepper.Stepper

	// Emergency stop, used as a motion inhibit flag
	eStopFlag bool

	// Motion in progress flag
	motionFlag bool

	// Axis homed flag
	homedFlag bool

	// Number of steps counted on the axis between the limit switches
	homingStepCount int

	// Menu busy flag
	menuBusyFlag bool
}

// List of items to pass:
//
// LCD
// Membrane 1, 2, 3, 4
// Red button, green button
// Left limit, right limit
func Initialize(lcd *goLCD20x4.LCD20x4, gm1 *sysfsGPIO.IOPin, gm2 *sysfsGPIO.IOPin, gm3 *sysfsGPIO.IOPin,
	gm4 *sysfsGPIO.IOPin, grb *sysfsGPIO.IOPin, ggb *sysfsGPIO.IOPin, gll *sysfsGPIO.IOPin, grl *sysfsGPIO.IOPin,
	stepper *softStepper.Stepper) PlateGenie {

	var pg PlateGenie

	pg.eStopFlag = true
	pg.motionFlag = false
	pg.homedFlag = false
	pg.menuBusyFlag = false

	// Set up the display
	lcd.FunctionSet(1, 1, 0)
	lcd.DisplayOnOffControl(1, 0, 0)
	lcd.EntryModeSet(1, 0)

	lcd.ClearDisplay()

	lcd.WriteLine("Welcome to", 1)
	lcd.WriteLine("PLATE GENIE", 2)

	pg.lcd = lcd

	m := CreateMenu(lcd)

	// ---------------
	// FIRST MENU ITEM
	// ---------------
	mi1 := m.AddMenuItem("Home Both", "", "", "   GO  ", "  GO   ")
	a1 := mi1.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a1 {
			case 1:
				fmt.Println("Home both")
				pg.homeBoth()
			case 2:
				fmt.Println("Home both")
				pg.homeBoth()
			}
		}
	}()

	// ----------------
	// SECOND MENU ITEM
	// ----------------
	mi2 := m.AddMenuItem("Home Single", "(This will unhome", "both axes.)", " Left  ", " Right ")
	a2 := mi2.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a2 {
			case 1:
				fmt.Println("Home left")
				pg.homeLeft()
			case 2:
				fmt.Println("Home right")
				pg.homeRight()
			}
		}
	}()

	// ---------------
	// THIRD MENU ITEM
	// ---------------
	mi3 := m.AddMenuItem("Move to Center", "", "", "   GO  ", "  GO   ")
	a3 := mi3.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a3 {
			case 1:
				fmt.Println("Move to center")
				pg.moveTrapezoidal(2000, 100, 90)
			case 2:
				fmt.Println("Move to center")
				pg.moveTrapezoidal(-2000, 100, 50)
			}
		}
	}()

	// ----------------
	// FOURTH MENU ITEM
	// ----------------
	mi4 := m.AddMenuItem("Speed", "(% Max Speed)", "100%", "   INC ", " DEC   ")
	a4 := mi4.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a4 {
			case 1:
				fmt.Println("Increase max speed")
			case 2:
				fmt.Println("Decrease max speed")
			}
		}
	}()

	// ---------------
	// FIFTH MENU ITEM
	// ---------------
	mi5 := m.AddMenuItem("Travel", "(% Max Distance)", "100%", "   INC ", " DEC   ")
	a5 := mi5.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a5 {
			case 1:
				fmt.Println("Increase travel distance")
			case 2:
				fmt.Println("Decrease travel distance")
			}
		}
	}()

	// ---------------
	// SIXTH MENU ITEM
	// ---------------
	mi6 := m.AddMenuItem("Stepper Hold", "", "", "   ENA ", " DIS   ")
	a6 := mi6.AddAction()
	// Action handler
	go func() {
		for {
			switch <-a6 {
			case 1:
				fmt.Println("Enable stepper hold")
				pg.stepper.EnableHold()
			case 2:
				fmt.Println("Disable stepper hold")
				pg.stepper.DisableHold()
			}
		}
	}()

	// Set up the membrane keypad GPIO here. Presume that the caller provides an input pin.
	gm1.SetTriggerEdge("rising")
	gm1.AddPinInterrupt()
	pg.gpioMembrane1 = gm1

	gm2.SetTriggerEdge("rising")
	gm2.AddPinInterrupt()
	pg.gpioMembrane2 = gm2

	gm3.SetTriggerEdge("rising")
	gm3.AddPinInterrupt()
	pg.gpioMembrane3 = gm3

	gm4.SetTriggerEdge("rising")
	gm4.AddPinInterrupt()
	pg.gpioMembrane4 = gm4

	// Red and green buttons
	grb.SetTriggerEdge("rising")
	grb.AddPinInterrupt()
	pg.gpioRedButton = grb

	ggb.SetTriggerEdge("rising")
	ggb.AddPinInterrupt()
	pg.gpioGreenButton = ggb

	// Left and right limit switches. Presume that pull-ups are defined in the DTO.
	gll.SetTriggerEdge("both")
	gll.AddPinInterrupt()
	pg.gpioLeftLimit = gll

	grl.SetTriggerEdge("both")
	grl.AddPinInterrupt()
	pg.gpioRightLimit = grl

	// Expend the events created with AddPinInterrupt()
	for k := 0; k < 8; k++ {
		fmt.Println("Expending", k, <-sysfsGPIO.GetInterruptStream())
	}

	go func() {
		for {
			s := <-sysfsGPIO.GetInterruptStream()
			switch s.IOPin.GPIONum {
			// Button 1
			case pg.gpioMembrane1.GPIONum:
				fmt.Println("Button 1 pressed")
				if !pg.menuBusyFlag {
					pg.menuBusyFlag = true
					go func() {
						m.Button1Pressed()
						pg.menuBusyFlag = false
					}()
				}
			// Button 2
			case pg.gpioMembrane2.GPIONum:
				if !pg.menuBusyFlag {
					pg.menuBusyFlag = true
					go func() {
						m.Button2Pressed()
						pg.menuBusyFlag = false
					}()
				}
			// Button 3
			case pg.gpioMembrane3.GPIONum:
				if !pg.menuBusyFlag {
					pg.menuBusyFlag = true
					go func() {
						m.Button3Pressed()
						pg.menuBusyFlag = false
					}()
				}
			// Button 4
			case pg.gpioMembrane4.GPIONum:
				if !pg.menuBusyFlag {
					pg.menuBusyFlag = true
					go func() {
						m.Button4Pressed()
						pg.menuBusyFlag = false
					}()
				}
			case pg.gpioLeftLimit.GPIONum:
				fmt.Println("Left limit hit")
			case pg.gpioRightLimit.GPIONum:
				fmt.Println("Right limit hit")
			case pg.gpioGreenButton.GPIONum:
				pg.eStopFlag = false
				pg.motionFlag = false
				fmt.Println("Green button hit")
			case pg.gpioRedButton.GPIONum:
				pg.eStopFlag = true
				pg.motionFlag = false
				fmt.Println("Red button hit")
			}
		}
	}()

	pg.stepper = stepper

	//	pg.homeBoth()

	for {
		time.Sleep(time.Second)
	}

	return pg

}

func agitate() {

}
