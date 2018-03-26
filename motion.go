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
	//	"math"
	"time"

	"github.com/the-sibyl/softStepper"
)

func (pg *PlateGenie) move(s *softStepper.Stepper, numStepsSigned int, speedPercentage int) error {
	if pg.motionFlag {
		return errors.New("Axis is already in motion")
	}

	if numStepsSigned < 0 {
		for k := 0; k < -numStepsSigned; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped due to emergency stop signal")
			}
			s.StepBackward()
		}
	} else if numStepsSigned > 0 {
		for k := 0; k < numStepsSigned; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped due to emergency stop signal")
			}
			s.StepForward()
		}
	}

	return nil
}

// Move with a trapezoidal acceleration profile
// numSteps: total number of steps to move
// speedPercentage: percentage of max stepper speed to move
// constantSpeedPercentage: percentage of time spent at constant speed
func (pg *PlateGenie) moveTrapezoidal(numStepsSigned int, speedPercentage int, constantSpeedPercentage int) error {
	if pg.motionFlag {
		return errors.New("Axis is already in motion")
	}

	if speedPercentage < 1 || speedPercentage > 100 {
		return errors.New("Invalid speed percentage parameter")
	} else if constantSpeedPercentage < 1 || constantSpeedPercentage > 99 {
		return errors.New("Invalid constant speed percentage parameter")
	}

	numSteps := 0
	forwardDirection := true
	if numStepsSigned < 0 {
		numSteps = -numStepsSigned
		forwardDirection = false
	} else if numStepsSigned > 0 {
		numSteps = numStepsSigned
	} else {
		// Do nothing as no motion was requested
		return nil
	}

	// Pulse duration from the stepper itself
	pulseDuration := pg.stepper.GetPulseDuration()
	// Delay added to slow down the stepper by the speedPeercentage parameter
	constantSpeedDelta := pulseDuration * time.Duration(100/float32(constantSpeedPercentage)-1)
	// Amount of total time taken per step at constant speed: stepper time AND speedPercentage slow-down time
	// are both included
	constantSpeedDelay := pulseDuration + constantSpeedDelta

	// I derived this equation on paper. The assumption that I made is that the average velocity of the trapezoidal
	// ramps is half the constant velocity.

	numStepsAccelDecel := int(float32(numSteps) / (2/(100/float32(constantSpeedPercentage)-1) + 1))
	fmt.Println(numStepsAccelDecel)
	numStepsAccel := numStepsAccelDecel / 2
	numStepsDecel := numStepsAccelDecel - numStepsAccel
	numStepsConstantSpeed := numSteps - numStepsAccel - numStepsDecel

	fmt.Println(numStepsAccel, numStepsDecel, numStepsConstantSpeed)

	// Actual acceleration time
	accelTime := time.Duration(numStepsAccel) * 2 * constantSpeedDelay
	// Mininum acceleration time based on the stepper speed
	minAccelTime := time.Duration(numStepsAccel) * pulseDuration
	// Amount of sleep time difference between two acceleration steps (accumulate)
	accelDelta := (accelTime - minAccelTime) / time.Duration(numStepsAccel*numStepsAccel)
	// Start value for the loop
	currentAccelSleepTime := constantSpeedDelta + accelDelta*time.Duration(numStepsAccel)

	for k := 0; k < numStepsAccel; k++ {
		if pg.eStopFlag {
			return errors.New("Motion stopped due to emergency stop signal")
		}
		if forwardDirection {
			pg.stepper.StepForward()
		} else {
			pg.stepper.StepBackward()
		}
		time.Sleep(currentAccelSleepTime)
		currentAccelSleepTime -= accelDelta
	}

	for k := 0; k < numStepsConstantSpeed; k++ {
		if pg.eStopFlag {
			return errors.New("Motion stopped due to emergency stop signal")
		}
		if forwardDirection {
			pg.stepper.StepForward()
		} else {
			pg.stepper.StepBackward()
		}
		time.Sleep(constantSpeedDelta)
	}

	// Copy the same values from the acceleration calculations
	decelDelta := accelDelta
	// Start value for the loop
	currentDecelSleepTime := constantSpeedDelta

	for k := 0; k < numStepsDecel; k++ {
		if pg.eStopFlag {
			return errors.New("Motion stopped due to emergency stop signal")
		}
		if forwardDirection {
			pg.stepper.StepForward()
		} else {
			pg.stepper.StepBackward()
		}
		time.Sleep(currentDecelSleepTime)
		currentDecelSleepTime += decelDelta
	}

	return nil
}

func (pg *PlateGenie) homeBoth() error {
	if pg.motionFlag {
		return errors.New("Axis is already in motion")
	}

	leftStatus, _ := pg.gpioLeftLimit.Read()
	rightStatus, _ := pg.gpioRightLimit.Read()

	// Measured number of steps between the two limit switches, set below
	homingStepCount := 0

	// The carriage is touching the left limit switch and needs to be backed off a bit for the first measurement
	if leftStatus == 1 && rightStatus == 0 {
		for k := 0; k < backoffSteps; k++ {
			pg.stepper.StepForward()
			rightStatus, _ = pg.gpioRightLimit.Read()
			// This case should only happen if backoffSteps is unnecesarily large or the carriage
			// separation is huge
			if rightStatus == 1 {
				break
			}
		}
	}

	leftStatus, _ = pg.gpioLeftLimit.Read()

	if leftStatus == 0 {
		for k := 0; k < maxHomingSteps; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped because of E-Stop")
			}
			pg.stepper.StepBackward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			if leftStatus == 1 {
				break
			}
			if k == maxHomingSteps-1 {
				return errors.New("Maximum number of steps exceeded on first left movement")
			}
		}
		for k := 0; k < maxHomingSteps; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped because of E-Stop")
			}
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
			if k == maxHomingSteps-1 {
				return errors.New("Maximum number of steps exceeded on left movement")
			}
		}
		for k := 0; k < homingStepCount/2; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped because of E-Stop")
			}
			pg.stepper.StepBackward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			if leftStatus == 1 {
				break
			}
			if k == maxHomingSteps-1 {
				return errors.New("Maximum number of steps exceeded on second left movement")
			}
		}
	} else {
		return errors.New(`Homing malfunction. Try increasing the number of backoff steps or beginning the homing operation
			 closer to the center.`)
	}

	pg.homingStepCount = homingStepCount
	pg.homedFlag = true

	return nil
}

func (pg *PlateGenie) homeLeft() error {
	if pg.motionFlag {
		return errors.New("Axis is already in motion")
	}

	pg.homedFlag = false

	leftStatus, _ := pg.gpioLeftLimit.Read()

	if leftStatus == 0 {
		for k := 0; k < maxHomingSteps; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped because of E-Stop")
			}
			pg.stepper.StepBackward()
			time.Sleep(time.Microsecond * homingStepDelay)
			leftStatus, _ = pg.gpioLeftLimit.Read()
			if leftStatus == 1 {
				break
			}
			if k == maxHomingSteps-1 {
				return errors.New("Maximum number of steps exceeded on left homing movement")
			}
		}
	}

	// Do this open-loop. backoffSteps should be on the order of the amount of steps required to clear the
	// limit switch.
	for k := 0; k < backoffSteps; k++ {
		pg.stepper.StepForward()
		time.Sleep(time.Microsecond * homingStepDelay)
	}

	return nil
}

func (pg *PlateGenie) homeRight() error {
	if pg.motionFlag {
		return errors.New("Axis is already in motion")
	}

	pg.homedFlag = false

	rightStatus, _ := pg.gpioRightLimit.Read()

	if rightStatus == 0 {
		for k := 0; k < maxHomingSteps; k++ {
			if pg.eStopFlag {
				return errors.New("Motion stopped because of E-Stop")
			}
			pg.stepper.StepForward()
			time.Sleep(time.Microsecond * homingStepDelay)
			rightStatus, _ = pg.gpioRightLimit.Read()
			if rightStatus == 1 {
				break
			}
			if k == maxHomingSteps-1 {
				return errors.New("Maximum number of steps exceeded on right homing movement")
			}
		}
	}

	// Do this open-loop. backoffSteps should be on the order of the amount of steps required to clear the
	// limit switch.
	for k := 0; k < backoffSteps; k++ {
		pg.stepper.StepBackward()
		time.Sleep(time.Microsecond * homingStepDelay)
	}

	return nil
}
