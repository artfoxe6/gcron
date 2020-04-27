package gcron

import (
	"fmt"
	"testing"
	"time"
)

func TestCron(t *testing.T) {

	ts := time.Now()
	//ts.Ad
	fmt.Println(int(ts.Weekday()))

}
