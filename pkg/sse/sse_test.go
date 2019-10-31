package sse

import (
	"bytes"
	"testing"
)

func TestEventAddDataFromLine(t *testing.T) {
	cases := []struct {
		line      string
		in        Event
		out       Event
		completed bool
		err       bool
	}{
		{"", Event{}, Event{}, true, false},
		{":sdfaasdfasdf", Event{}, Event{}, false, false},
		{"id:  4358398475 ", Event{}, Event{"", "4358398475", nil}, false, false},
		{"event:  rc ", Event{}, Event{"rc", "", nil}, false, false},
		{`data:  {"a": 1,\\n "b": "c"} `, Event{}, Event{"", "", []byte(`{"a": 1,\\n "b": "c"}`)}, false, false},
		{`data:  line2`, Event{"", "", []byte("line1")}, Event{"", "", []byte("line1\nline2")}, false, false},
		{"dflgkjkljdfsgkljs", Event{}, Event{}, false, true},
	}

	for _, c := range cases {
		e := c.in

		out, err := e.addDataFromLine(c.line)
		if c.err && err == nil {
			t.Errorf("Event{}.addDataFromLine(%s) should have errored but did not", c.line)
		}

		if !c.err && err != nil {
			t.Errorf("Event{}.addDataFromLine(%s) should not have errored but did: %s", c.line, err)
		}

		if c.completed && !out {
			t.Errorf("Event{}.addDataFromLine(%s) should have reported completion but did not", c.line)
		}

		if !c.completed && out {
			t.Errorf("Event{}.addDataFromLine(%s) should not have reported completion but did", c.line)
		}

		if e.ID != c.out.ID {
			t.Errorf("Event{}.addDataFromLine(%s) should have put ID = %s but ID was %s", c.line, c.out.ID, e.ID)
		}

		if e.Event != c.out.Event {
			t.Errorf("Event{}.addDataFromLine(%s) should have put Event = %s but Event was %s", c.line, c.out.Event, e.Event)
		}

		if !bytes.Equal(e.Data, c.out.Data) {
			t.Errorf("Event{}.addDataFromLine(%s) should have put Data = %s but Data was %s", c.line, string(c.out.Event), string(e.Event))
		}
	}
}

func TestStreamInvalidCases(t *testing.T) {
	cases := []struct {
		url string
		err bool
	}{
		{"", true},
		{"http://httpstat.us/400", true},
	}

	for _, c := range cases {
		channel := make(chan Event, 1)
		cancel, _, err := Stream(channel, nil, c.url, nil)

		if c.err && err == nil {
			t.Errorf("Stream(%s) should have errored but did not", c.url)
		}

		if !c.err && err != nil {
			t.Errorf("Stream(%s) should not have errored but did: %s", c.url, err)
		}

		cancel()
	}
}
