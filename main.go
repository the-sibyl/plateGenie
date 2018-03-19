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
	"math"
	"time"

	"./menu"

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

// Distance is in millimeters. The sign connotes direction.
func move(s *softStepper.Stepper, dist float64) {
	if math.Abs(dist) > float64(maxDistance) {
		panic("Requested distance exceeds safe travel limits.")
	}

	numSteps := int(math.Floor(math.Abs((float64(dist))/screwLead) * stepsPerRev))

	if dist < 0 {
		s.StepForwardMulti(numSteps)
	} else if dist > 0 {
		s.StepBackwardMulti(numSteps)
	}
}

// Move with a trapezoidal acceleration profile
func moveTrapezoidal(s *softStepper.Stepper, dist float64) {
	if math.Abs(dist) > float64(maxDistance) {
		panic("Requested distance exceeds safe travel limits.")
	}

	averageDelay := float64(trapMaximumDelay+trapMinimumDelay) / 2.0
	numDelayDivisions := int(math.Floor(trapAccelPeriod / averageDelay))
	delayIncrement := float64(trapMaximumDelay-trapMinimumDelay) / float64(numDelayDivisions)

	fmt.Println(averageDelay, numDelayDivisions, delayIncrement)

	// Accelerate
	stepCountAcc := 0
	for delay := float64(trapMaximumDelay); delay >= float64(trapMinimumDelay); delay = delay - delayIncrement {
		if stepCountAcc > numDelayDivisions {
			panic("Trapezoidal acceleration error")
		}
		if dist < 0 {
			s.StepForward()
		} else if dist > 0 {
			s.StepBackward()
		}
		time.Sleep(time.Microsecond * time.Duration(delay))
		stepCountAcc++
	}

	// Constant speed

	numStepsConstantSpeed := int(math.Floor(math.Abs((float64(dist))/screwLead)*stepsPerRev)) - 2*stepCountAcc

	// FIXME: Enforce that the acceleration and deceleration segments do not cause the carriage to travel too far!
	if numStepsConstantSpeed < 0 {
		panic("Acceleration/deceleration profile is too long")
	}

	if dist < 0 {
		s.StepForwardMulti(numStepsConstantSpeed)
	} else if dist > 0 {
		s.StepBackwardMulti(numStepsConstantSpeed)
	}

	// Decelerate
	stepCountDec := 0
	for delay := float64(trapMinimumDelay); delay <= float64(trapMaximumDelay); delay = delay + delayIncrement {
		if stepCountDec > numDelayDivisions {
			panic("Trapezoidal acceleration error")
		}
		if dist < 0 {
			s.StepForward()
		} else if dist > 0 {
			s.StepBackward()
		}
		time.Sleep(time.Microsecond * time.Duration(delay))
		stepCountDec++
	}
	fmt.Println("Accel Steps:", stepCountAcc)
	fmt.Println("Decel Steps:", stepCountDec)
	if stepCountAcc != stepCountDec {
		panic(`Mismatch between acceleration and deceleration steps. 
			This would cause the carriage to drift in one direction.`)
	}

}

// Add a homing function with a timeout

// Count the number of steps from left side to right side

// Add parameters for the following
// 	Stepper speed
// 	Leadscrew lead and/or pitch
// 	Steps per revolution
// 	Agitation distance
//	Agitation timer

func main() {
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
//	m.AddMenuItem("       Speed        ", "    (% Max Speed)   ", "100", "   INC ", " DEC   ")
//	m.AddMenuItem("       Travel       ", "  (% Max Distance)  ", "10", "   INC ", " DEC   ")
	m.AddMenuItem("Speed", "(% Max Speed)", "100%", "   INC ", " DEC   ")
	m.AddMenuItem("Travel", "(% Max Distance)", "100%", "   INC ", " DEC   ")
	go func() {
		for {
		m.Next()
		time.Sleep(time.Second * 1)
		}

	} ()

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

/*
	go func() {
		for {
			select {
				case s := <-sysfsGPIO.GetInterruptStream():
					switch(s.IOPin.GPIONum) {
						case 19:
							lcd.WriteLine("Button 1 pressed last", 4)
						case 26:
							lcd.WriteLine("Button 2 pressed last", 4)
						case 6:
							lcd.WriteLine("Button 3 pressed last", 4)
						case 13:
							lcd.WriteLine("Button 4 pressed last", 4)
					}
			}
		}
	} ()
*/

	stepper1 := softStepper.InitStepper(18, 23, 24, 25, 8, time.Microsecond*stepperSpeed)
	defer stepper1.ReleaseStepper()

	stepper1.EnableHold()

	for {
		moveTrapezoidal(stepper1, 60.105)
		moveTrapezoidal(stepper1, -60.105)
		/*
			move(stepper1, -0.5)
			move(stepper1, 0.5)
		*/
	}

}

func home() {

}

func agitate() {

}
