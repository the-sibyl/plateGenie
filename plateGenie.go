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

type PlateGenie struct {
	lcd *goLCD20x4.LCD20x4

	gpioRedButton *sysfsGPIO.IOPin
	gpioGreenButton *sysfsGPIO.IOPin

	gpioLeftLimit *sysfsGPIO.IOPin
	gpioRightLimit *sysfsGPIO.IOPin

	stepper *softStepper.Stepper
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
