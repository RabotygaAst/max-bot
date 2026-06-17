package service

import (
	"testing"
	"time"
)

func TestParseInvoicePeriod(t *testing.T) {
	period, chooser := parseInvoicePeriod("квитанция", time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC))
	if period != "2026-06" || chooser {
		t.Fatalf("got %s chooser=%v", period, chooser)
	}
	period, chooser = parseInvoicePeriod("счет 2026-05", time.Now())
	if period != "2026-05" || chooser {
		t.Fatalf("got %s chooser=%v", period, chooser)
	}
}

func TestParseAppealCommand(t *testing.T) {
	cat, text := parseAppealCommand("авария прорвало трубу", "")
	if cat != "emergency" || text != "прорвало трубу" {
		t.Fatalf("got %s %q", cat, text)
	}
	cat, text = parseAppealCommand("жалоба некачественная уборка", "")
	if cat != "complaint" || text != "некачественная уборка" {
		t.Fatalf("got %s %q", cat, text)
	}
}

func TestParseAppointmentTopic(t *testing.T) {
	if got := parseAppointmentTopic("запись billing"); got != "billing" {
		t.Fatalf("got %q", got)
	}
}

func TestParseReadingCommand(t *testing.T) {
	meter, value, ok := parseReadingCommand("показание MTR-001 245.678")
	if !ok || meter != "MTR-001" || value != 245.678 {
		t.Fatalf("got %q %.3f %v", meter, value, ok)
	}
	_, _, ok = parseReadingCommand("показание MTR-001 -1")
	if ok {
		t.Fatal("negative value must be invalid")
	}
}
