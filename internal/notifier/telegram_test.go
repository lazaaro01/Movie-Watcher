package notifier

import "testing"

func TestSendMessageWithButtonsDisabledFallback(t *testing.T) {
	n := &Notifier{enabled: false}

	err := n.SendMessageWithButtons("texto de teste", []Button{
		{Label: "18:30", URL: "https://example.com/comprar"},
	})
	if err != nil {
		t.Fatalf("expected no error when notifier is disabled, got: %v", err)
	}
}

func TestSendMessageWithButtonsDisabledNoButtons(t *testing.T) {
	n := &Notifier{enabled: false}

	if err := n.SendMessageWithButtons("texto de teste", nil); err != nil {
		t.Fatalf("expected no error with empty buttons, got: %v", err)
	}
}
