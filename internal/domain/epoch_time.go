package domain

import (
	"encoding/json"
	"time"
)

type EpochTime float64

func (e *EpochTime) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*e = EpochTime(v)
	return nil
}

func (e EpochTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(e))
}

func (e EpochTime) Time() time.Time {
	sec := int64(e)
	nsec := int64((float64(e) - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC()
}

func EpochTimeFromTime(t time.Time) EpochTime {
	return EpochTime(float64(t.UnixNano()) / 1e9)
}
