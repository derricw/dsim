package model

import (
	"testing"
	"time"
)

func TestProcessBasic(t *testing.T) {
	inReq := 5
	outCount := 2
	inch := make(Pool, 100)
	inputChans := map[Pool]int{
		inch: inReq,
	}
	outch := make(Pool, 100)
	outputChans := map[Pool]int{
		outch: outCount,
	}

	p := NewProcess(
		inputChans,
		outputChans,
		time.Second*2,
		"TestProcess",
	)

	go p.Run(time.Second * 10)

	t0 := time.Now()
	for i := 0; i < inReq; i++ {
		inch <- time.Now()
	}

	for i := 0; i < outCount; i++ {
		select {
		case tOut := <-outch:
			if tOut.Sub(t0) < p.duration {
				t.Errorf("process didn't wait full duration!")
			}
		case <-time.After(time.Second * 1):
			t.Errorf("failed to receive process output")
		}
	}

}

func TestProcessProducer(t *testing.T) {
	outCount := 2
	outch := make(Pool, 100)
	outputChans := map[Pool]int{
		outch: outCount,
	}

	p := NewProcess(
		nil,
		outputChans,
		time.Second*2,
		"TestProcess",
	)

	go p.Run(time.Second * 10)

	t0 := time.Now()
	// first batch should be sent immediately
	for i := 0; i < outCount; i++ {
		select {
		case tOut := <-outch:
			if tOut.Sub(t0) > p.duration {
				t.Errorf("producer took to long to finish first batch!")
			}
		case <-time.After(time.Second * 1):
			t.Errorf("failed to receive process output")
		}
	}
	// second batch should come just after `duration`
	for i := 0; i < outCount; i++ {
		select {
		case tOut := <-outch:
			if tOut.Sub(t0) < p.duration {
				t.Errorf("producer didn't send the second batch after `duration`!")
			}
		case <-time.After(time.Second * 1):
			t.Errorf("failed to receive process output")
		}
	}

}

func TestProcessTwoInputs(t *testing.T) {
	inReq := 3
	outCount := 2
	inch0 := make(Pool, 100)
	inch1 := make(Pool, 100)
	inputChans := map[Pool]int{
		inch0: inReq,
		inch1: inReq + 1,
	}
	outch := make(Pool, 100)
	outputChans := map[Pool]int{
		outch: outCount,
	}

	p := NewProcess(
		inputChans,
		outputChans,
		time.Second*2,
		"TestProcess",
	)

	go p.Run(time.Second * 10)

	t0 := time.Now()
	for i := 0; i < inReq; i++ {
		inch0 <- time.Now()
		inch1 <- time.Now()
	}

	// should not fire batch yet
	for i := 0; i < outCount; i++ {
		select {
		case _ = <-outch:
			t.Errorf("fired an output before receiving all inputs!")
		case <-time.After(time.Millisecond * 10):
		}
	}

	inch1 <- time.Now()
	// now we should see one
	select {
	case tOut := <-outch:
		if tOut.Sub(t0) < p.duration {
			t.Errorf("process didn't wait full duration!")
		}
	case <-time.After(time.Second * 1):
		t.Errorf("failed to receive process output")
	}
}
