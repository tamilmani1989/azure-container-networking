package requestcontroller

import (
	"testing"
	"time"
)

func TestWithin(t *testing.T) {
	inOneHour := time.Now().Add(time.Hour)

	//inOneHour should be within 2 hours
	w := within(inOneHour, time.Hour*2)

	if !w {
		t.Fatalf("expected %v, got %v", true, w)
	}

	inThreeHours := time.Now().Add(3 * time.Hour)

	//inThreeHours should not be within 2 hours
	w2 := within(inThreeHours, time.Hour*2)

	if w2 {
		t.Fatalf("expected %v, got %v", false, w2)
	}
}
