package logger

import (
	"testing"
	"time"
)

func TestFormatterSampleExecution(t *testing.T) {
	f := NewFormatter("2006-01-02 15:04:05,000")
	tests := []struct {
		msg  Message
		want string
	}{
		{
			msg: Message{
				Level: INFO, Content: "Empty txnIds Nothing to fetch",
				Timestamp: time.Date(2022, 6, 27, 11, 14, 44, 942_000_000, time.UTC),
			},
			want: "INFO [2022-06-27 11:14:44,942] Empty txnIds Nothing to fetch",
		},
		{
			msg: Message{
				Level: WARN, Content: "No user found for the phone number",
				Timestamp: time.Date(2022, 6, 27, 11, 28, 6, 229_000_000, time.UTC),
			},
			want: "WARN [2022-06-27 11:28:06,229] No user found for the phone number",
		},
	}
	for _, tt := range tests {
		if got := f.Format(tt.msg); got != tt.want {
			t.Errorf("got  %q\nwant %q", got, tt.want)
		}
	}
}
