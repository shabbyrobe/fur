package furball

import (
	"strconv"
	"time"
)

// Duration provides a time.Duration that marshals to/from a float
// representing milliseconds.
type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) MarshalJSON() ([]byte, error) {
	fv := float64(d) / float64(time.Millisecond)
	fs := strconv.FormatFloat(fv, 'f', 9, 64)
	return []byte(fs), nil
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	fv, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	fd := time.Duration(fv * float64(time.Millisecond))
	*d = Duration(fd)
	return nil
}
