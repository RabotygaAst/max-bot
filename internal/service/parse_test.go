package service

import (
	"testing"
	"time"
)

func TestParseInvoicePeriod(t *testing.T) {
	now := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	if got, help := parseInvoicePeriod("квитанция", now); got != "2026-06" || help {
		t.Fatalf("got %q help=%v", got, help)
	}
	if got, help := parseInvoicePeriod("квитанция 2026-05", now); got != "2026-05" || help {
		t.Fatalf("got %q help=%v", got, help)
	}
	if _, help := parseInvoicePeriod("квитанция май", now); !help {
		t.Fatalf("invalid period should request help")
	}
}

func TestParseReadingCommand(t *testing.T) {
	if id, v, ok := parseReadingCommand("показание MTR-001 245.678"); !ok || id != "MTR-001" || v != 245.678 {
		t.Fatalf("dot parse failed: %s %f %v", id, v, ok)
	}
	if id, v, ok := parseReadingCommand("показание MTR-001 245,678"); !ok || id != "MTR-001" || v != 245.678 {
		t.Fatalf("comma parse failed: %s %f %v", id, v, ok)
	}
	if _, _, ok := parseReadingCommand("показание MTR-001 -1"); ok {
		t.Fatalf("negative accepted")
	}
	if _, _, ok := parseReadingCommand("показание MTR-001 abc"); ok {
		t.Fatalf("non-number accepted")
	}
}

func TestParseAppointmentTopic(t *testing.T) {
	if got := parseAppointmentTopic("запись"); got != "" {
		t.Fatalf("expected empty topic, got %q", got)
	}
	if got := parseAppointmentTopic("запись billing"); got != "billing" {
		t.Fatalf("expected billing, got %q", got)
	}
}

func TestParseAppealCommand(t *testing.T) {
	cases := []struct{ in, cat, text string }{{"обращение текст", "general", "текст"}, {"заявка текст", "general", "текст"}, {"авария текст", "emergency", "текст"}, {"жалоба текст", "complaint", "текст"}}
	for _, tc := range cases {
		cat, text := parseAppealCommand(tc.in, "")
		if cat != tc.cat || text != tc.text {
			t.Fatalf("%q => %q %q", tc.in, cat, text)
		}
	}
}
