package sse

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Event struct {
	Event string
	ID    string
	Data  []byte
}

func (e *Event) addDataFromLine(line string) (bool, error) {
	switch {
	// signal event completion
	case len(line) == 0:
		return true, nil

	// ignore comment
	case strings.HasPrefix(line, ":"):
		return false, nil

	// id of event
	case strings.HasPrefix(line, "id:"):
		e.ID = strings.TrimSpace(line[3:])
		return false, nil

	// name of event
	case strings.HasPrefix(line, "event:"):
		e.Event = strings.TrimSpace(line[6:])
		return false, nil

	// data
	case strings.HasPrefix(line, "data:"):
		data := strings.TrimSpace(line[5:])
		if e.Data != nil {
			data = "\n" + data
			e.Data = append(e.Data, []byte(data)...)
		} else {
			e.Data = []byte(data)
		}
		return false, nil

	default:
		return false, fmt.Errorf("line didn't match id, event, or data: %s", line)
	}
}

// Takes ownership of eventChan (will handle closing)
func Stream(eventChan chan<- Event, wg *sync.WaitGroup, url string, client *http.Client) (context.CancelFunc, <-chan error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		close(errChan)
		close(eventChan)
		return cancel, errChan, err
	}

	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		close(errChan)
		close(eventChan)
		return cancel, errChan, err
	}

	if resp.StatusCode != 200 {
		close(errChan)
		close(eventChan)
		return cancel, errChan, fmt.Errorf("sse url returned non-200 code: %d", resp.StatusCode)
	}

	go stream(wg, resp, eventChan, errChan)
	return cancel, errChan, nil
}

func stream(wg *sync.WaitGroup, resp *http.Response, eventChan chan<- Event, errChan chan<- error) {
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}

	defer resp.Body.Close()
	defer close(eventChan)
	defer close(errChan)

	reader := bufio.NewReader(resp.Body)
	event := Event{}

	for {
		raw, err := reader.ReadString('\n')
		if err != nil {
			errChan <- err
			return
		}

		completed, err := event.addDataFromLine(strings.TrimSpace(raw))
		if err != nil {
			errChan <- err
			return
		}

		if completed {
			eventChan <- event
			event = Event{}
		}
	}
}
