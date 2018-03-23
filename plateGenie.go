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
	"errors"
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
)

type PlateGenie struct {
	lcd *goLCD20x4.LCD20x4

	gpioMembrane1 *sysfsGPIO.IOPin
	gpioMembrane2 *sysfsGPIO.IOPin
	gpioMembrane3 *sysfsGPIO.IOPin
	gpioMembrane4 *sysfsGPIO.IOPin

	gpioRedButton *sysfsGPIO.IOPin
	gpioGreenButton *sysfsGPIO.IOPin

	gpioLeftLimit *sysfsGPIO.IOPin
	gpioRightLimit *sysfsGPIO.IOPin

	stepper *softStepper.Stepper
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

	// Set up the display
	lcd.FunctionSet(1, 1, 0)
	lcd.DisplayOnOffControl(1, 0, 0)
	lcd.EntryModeSet(1, 0)

	lcd.ClearDisplay()

	lcd.WriteLine("Welcome to", 1)
	lcd.WriteLine("PLATE GENIE", 2)

	pg.lcd = lcd

	m := CreateMenu(lcd)
	m.AddMenuItem("Home Both", "", "", "   GO  ", "  GO   ")
	m.AddMenuItem("Home Single", "", "", " Left  ", " Right ")
	m.AddMenuItem("Move to Center", "", "", "   GO  ", "  GO   ")
	m.AddMenuItem("Speed", "(% Max Speed)", "100%", "   INC ", " DEC   ")
	m.AddMenuItem("Travel", "(% Max Distance)", "100%", "   INC ", " DEC   ")

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
						case pg.gpioMembrane1.GPIONum:
							m.Prev()
							lcd.WriteLine("Button 1 pressed last", 4)
						// Button 2
						case pg.gpioMembrane2.GPIONum:
							lcd.WriteLine("Button 2 pressed last", 4)
						// Button 3
						case pg.gpioMembrane3.GPIONum:
							lcd.WriteLine("Button 3 pressed last", 4)
						// Button 4
						case pg.gpioMembrane4.GPIONum:
							lcd.WriteLine("Button 4 pressed last", 4)
							m.Next()
						case pg.gpioLeftLimit.GPIONum:
							fmt.Println("Left limit hit")
						case pg.gpioRightLimit.GPIONum:
							fmt.Println("Right limit hit")
						case pg.gpioGreenButton.GPIONum:
							fmt.Println("Green button hit")
						case pg.gpioRedButton.GPIONum:
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

	return pg

}
// Maximum number of steps to be traversed for an axis move on a homing operation
const maxHomingSteps = 100000
const homingStepDelay = 1000

func (pg *PlateGenie) homeBoth() error {
	leftStatus, _ := pg.gpioLeftLimit.Read()
	rightStatus, _ := pg.gpioRightLimit.Read()

	// Measured number of steps between the two limit switches, set below
	homingStepCount := 0

	if leftStatus == 0 {
		for k := 0; k < maxHomingSteps; k++ {
			pg.stepper.StepBackward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			if leftStatus == 1 {
				break
			}
			if k == maxHomingSteps - 1 {
				return errors.New("Maximum number of steps exceeded on first left movement")
			}
		}
		for k := 0; k < maxHomingSteps; k++ {
			pg.stepper.StepForward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			rightStatus, _ = pg.gpioRightLimit.Read()
			if leftStatus == 0 && rightStatus == 0 {
				homingStepCount++
			}
			if rightStatus == 1 {
				break
			}
			if k == maxHomingSteps - 1 {
				return errors.New("Maximum number of steps exceeded on left movement")
			}
		}
		for k := 0; k < homingStepCount / 2; k++ {
			pg.stepper.StepBackward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			if leftStatus == 1 {
				break
			}
			if k == maxHomingSteps - 1 {
				return errors.New("Maximum number of steps exceeded on second left movement")
			}
		}
	}

	return nil
}

func agitate() {

}
