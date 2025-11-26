package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiEvent_GetControlMessageDetails(t *testing.T) {
	controlMessage := ControlMessageDetails{
		CloseCode:    1000,
		MessageType:  1,
		CloseMessage: "test close",
	}

	event := ApiEvent{
		Type:                  "ADDED",
		Object:                "test-object",
		controlMessageDetails: controlMessage,
	}

	result := event.GetControlMessageDetails()

	assert.Equal(t, controlMessage, result)
	assert.Equal(t, 1000, result.CloseCode)
	assert.Equal(t, 1, result.MessageType)
	assert.Equal(t, "test close", result.CloseMessage)
}

func TestApiEventConstructor(t *testing.T) {
	controlMessage := ControlMessageDetails{
		CloseCode:    2000,
		MessageType:  2,
		CloseMessage: "constructor test",
	}

	event := ApiEventConstructor("MODIFIED", "test-obj", controlMessage)

	assert.Equal(t, "MODIFIED", event.Type)
	assert.Equal(t, "test-obj", event.Object)
	assert.Equal(t, controlMessage, event.controlMessageDetails)
}

func TestControlMessageDetails(t *testing.T) {
	details := ControlMessageDetails{
		CloseCode:    3000,
		MessageType:  3,
		CloseMessage: "details test",
	}

	assert.Equal(t, 3000, details.CloseCode)
	assert.Equal(t, 3, details.MessageType)
	assert.Equal(t, "details test", details.CloseMessage)
}

func TestHandler(t *testing.T) {
	// Create a mock channel
	ch := make(chan ApiEvent, 1)

	// Create a mock stop function
	stopCalled := false
	stopFunc := func() {
		stopCalled = true
	}

	handler := Handler{
		Channel:      (<-chan ApiEvent)(ch),
		StopWatching: stopFunc,
	}

	// Test the channel
	assert.NotNil(t, handler.Channel)

	// Test the stop function
	handler.StopWatching()
	assert.True(t, stopCalled)
}
