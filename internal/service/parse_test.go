package service

import (
	"strings"
	"testing"
	"time"

	"example.com/max-bot-go/internal/model"
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

func TestFormatMetersMessageHVSAndGVSTariffREK(t *testing.T) {
	msg := formatMetersMessage([]model.Meter{
		{ID: "M1", DisplayName: "ХВС", Unit: "м³", LastValue: 234.56, LastReadingDate: "2026-04-06", CanSubmit: true},
		{ID: "M2", DisplayName: "ГВС", Unit: "м³", LastValue: 128.4, LastReadingDate: "2026-04-06", CanSubmit: true},
		{ID: "M3", DisplayName: "Тариф РЭК", Unit: "кВт⋅ч", LastValue: 10543, LastReadingDate: "2026-04-06", CanSubmit: true},
	})
	for _, want := range []string{"ХВС: 234.560 м³ — 06.04.2026", "ГВС: 128.400 м³ — 06.04.2026", "Тариф РЭК: 10543.000 кВт⋅ч — 06.04.2026", "ХВС — M1", "ГВС — M2", "Тариф РЭК — M3"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message does not contain %q:\n%s", want, msg)
		}
	}
}

func TestFormatMetersMessageDoesNotInventMissingGVS(t *testing.T) {
	msg := formatMetersMessage([]model.Meter{
		{ID: "M1", DisplayName: "ХВС", Unit: "м³", LastValue: 234.56, LastReadingDate: "2026-04-06", CanSubmit: true},
		{ID: "M4", DisplayName: "Отопление", Unit: "Гкал", LastValue: 12, LastReadingDate: "2026-04-06", CanSubmit: true},
		{ID: "M3", DisplayName: "Тариф РЭК", Unit: "кВт⋅ч", LastValue: 10543, LastReadingDate: "2026-04-06", CanSubmit: true},
	})
	if strings.Contains(msg, "ГВС") {
		t.Fatalf("message invented missing GVS:\n%s", msg)
	}
	if !strings.Contains(msg, "Отопление: 12.000 Гкал — 06.04.2026") {
		t.Fatalf("message does not contain heating:\n%s", msg)
	}
}

func TestFormatMetersMessageCanSubmitFalseExcludesID(t *testing.T) {
	msg := formatMetersMessage([]model.Meter{{ID: "M1", DisplayName: "ХВС", Unit: "м³", LastValue: 234.56, LastReadingDate: "2026-04-06", CanSubmit: false, Reason: "Не найден прибор учета для передачи показаний"}})
	if !strings.Contains(msg, "Нельзя передать: Не найден прибор учета для передачи показаний") {
		t.Fatalf("reason missing:\n%s", msg)
	}
	if strings.Contains(msg, "ХВС — M1") {
		t.Fatalf("blocked meter ID should not be available:\n%s", msg)
	}
}

func TestFormatMetersMessageDisplayNameFallback(t *testing.T) {
	cases := []model.Meter{
		{ID: "M1", ServiceName: "Услуга", CanSubmit: true},
		{ID: "M2", TariffName: "Тариф", CanSubmit: true},
		{ID: "M3", Resource: "Ресурс", CanSubmit: true},
		{ID: "M4", CanSubmit: true},
	}
	msg := formatMetersMessage(cases)
	for _, want := range []string{"Услуга", "Тариф", "Ресурс", "M4"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("fallback %q missing:\n%s", want, msg)
		}
	}
}
