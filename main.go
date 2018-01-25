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
//	"fmt"
	"time"

	"github.com/the-sibyl/softStepper"
)

const (
	stepperSpeed = 5
	screwLead = 4.0
	stepsPerRev = 180
)



func main() {
	stepper1 := softStepper.InitStepper(18, 23, 24, 25, 8, time.Microsecond * 1000)
	stepper1.EnableHold()

	stepper1.StepBackwardMulti(1000)

	for {
		stepper1.StepForwardMulti(2000)
		stepper1.StepBackwardMulti(2000)
	}

}

func home() {

}

func agitate() {

}

// Add a homing function with a timeout

// Count the number of steps from left side to right side

// Add parameters for the following
// 	Stepper speed
// 	Leadscrew lead and/or pitch
// 	Steps per revolution
// 	Agitation distance
//	Agitation timer	
