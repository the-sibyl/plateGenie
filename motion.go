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
	"math"
	"time"

	"github.com/the-sibyl/softStepper"
)
// Distance is in millimeters. The sign connotes direction.
func move(s *softStepper.Stepper, dist float64) (int, error) {
	if math.Abs(dist) > float64(maxDistance) {
		return 0, errors.New("Requested distance exceeds safe travel limits.")
	}

	numSteps := int(math.Floor(math.Abs((float64(dist))/screwLead) * stepsPerRev))

	if dist < 0 {
		s.StepForwardMulti(numSteps)
	} else if dist > 0 {
		s.StepBackwardMulti(numSteps)
	}

	return numSteps, nil
}

// Move with a trapezoidal acceleration profile
func moveTrapezoidal(s *softStepper.Stepper, dist float64) (int, error) {
	if math.Abs(dist) > float64(maxDistance) {
		return 0, errors.New("Requested distance exceeds safe travel limits.")
	}

	averageDelay := float64(trapMaximumDelay+trapMinimumDelay) / 2.0
	numDelayDivisions := int(math.Floor(trapAccelPeriod / averageDelay))
	delayIncrement := float64(trapMaximumDelay-trapMinimumDelay) / float64(numDelayDivisions)

	fmt.Println(averageDelay, numDelayDivisions, delayIncrement)

	// Accelerate
	stepCountAcc := 0
	for delay := float64(trapMaximumDelay); delay >= float64(trapMinimumDelay); delay = delay - delayIncrement {
		if stepCountAcc > numDelayDivisions {
			return 0, errors.New("Trapezoidal acceleration error")
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
		return 0, errors.New("Acceleration/deceleration profile is too long")
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
			return 0, errors.New("Trapezoidal acceleration error")
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
		return 0, errors.New(`Mismatch between acceleration and deceleration steps. 
			This would cause the carriage to drift in one direction.`)
	}

	totalSteps := stepCountAcc + numStepsConstantSpeed + stepCountDec
	return totalSteps, nil
}