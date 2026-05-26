package wevent

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFormatInsert(t *testing.T) {
	got := Format(Event{Origin: 'K', Type: 'I', Q0: 10, Q1: 11, Text: "a"})
	want := []byte("KI10 11 0 1 a\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestFormatNewlineInsert(t *testing.T) {
	got := Format(Event{Origin: 'K', Type: 'I', Q0: 10, Q1: 11, Text: "\n"})
	want := []byte{'K', 'I', '1', '0', ' ', '1', '1', ' ', '0', ' ', '1', ' ', '\n', '\n'}
	if !bytes.Equal(got, want) {
		t.Fatalf("Format() = %#v, want %#v", got, want)
	}
}

func TestReadNewlineInsert(t *testing.T) {
	ev, err := Read(bufio.NewReader(bytes.NewReader([]byte("KI10 11 0 1 \n\n"))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Origin != 'K' || ev.Type != 'I' || ev.Q0 != 10 || ev.Q1 != 11 || ev.Flag != 0 || ev.Text != "\n" {
		t.Fatalf("event = %#v", ev)
	}
}

func TestReadMultilineText(t *testing.T) {
	ev, err := Read(bufio.NewReader(bytes.NewReader([]byte("KI10 21 0 11 hello\nworld\n"))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Text != "hello\nworld" {
		t.Fatalf("Text = %q, want %q", ev.Text, "hello\nworld")
	}
}

func TestReadPreservesSpacesAndTabs(t *testing.T) {
	ev, err := Read(bufio.NewReader(bytes.NewReader([]byte("KI0 6 0 6 a  b\tc\n"))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Text != "a  b\tc" {
		t.Fatalf("Text = %q, want %q", ev.Text, "a  b\tc")
	}
}

func TestReadAllowsReplacementChar(t *testing.T) {
	ev, err := Read(bufio.NewReader(bytes.NewReader([]byte("KI0 1 0 1 �\n"))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Text != "�" {
		t.Fatalf("Text = %q, want %q", ev.Text, "�")
	}
}

func TestReadRejectsInvalidUTF8(t *testing.T) {
	_, err := Read(bufio.NewReader(bytes.NewReader([]byte{'K', 'I', '0', ' ', '1', ' ', '0', ' ', '1', ' ', 0xff, '\n'})))
	if err == nil {
		t.Fatal("Read succeeded with invalid UTF-8")
	}
}

func TestReadDeletion(t *testing.T) {
	ev, err := Read(bufio.NewReader(bytes.NewReader([]byte("KD10 20 0 0 \n"))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Origin != 'K' || ev.Type != 'D' || ev.Q0 != 10 || ev.Q1 != 20 || ev.Flag != 0 || ev.Text != "" {
		t.Fatalf("event = %#v", ev)
	}
}

func TestReadFormatRoundTripWithNegativeFields(t *testing.T) {
	want := Event{Origin: 'K', Type: 'I', Q0: -1, Q1: -2, Flag: -3, Text: "x"}
	ev, err := Read(bufio.NewReader(bytes.NewReader(Format(want))))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev != want {
		t.Fatalf("event = %#v, want %#v", ev, want)
	}
}

func TestReadCleanEOFIsBare(t *testing.T) {
	_, err := Read(bufio.NewReader(bytes.NewReader(nil)))
	if err != io.EOF {
		t.Fatalf("err = %v, want bare io.EOF", err)
	}
}

func TestReadMidRecordTruncationIsWrappedEOF(t *testing.T) {
	// Origin consumed, type missing → mid-record truncation must wrap io.EOF
	// (errors.Is matches), but the bare io.EOF check used to gate clean
	// record boundaries must NOT match.
	_, err := Read(bufio.NewReader(bytes.NewReader([]byte{'K'})))
	if err == io.EOF {
		t.Fatal("mid-record truncation returned bare io.EOF; clean-boundary detection would swallow it")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want wrapped io.EOF", err)
	}
}

func TestReadRejectsOversizedNR(t *testing.T) {
	// NR = 2 billion would otherwise allocate ~8 GiB up front.
	line := []byte("KI0 0 0 2000000000 \n")
	_, err := Read(bufio.NewReader(bytes.NewReader(line)))
	if err == nil {
		t.Fatal("oversized NR should be rejected")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("err = %v, want out-of-range rejection", err)
	}
}

func TestReadRejectsOverlongInt(t *testing.T) {
	// 25 digits exceeds MaxIntDigits and would never fit a Go int anyway.
	long := strings.Repeat("9", MaxIntDigits+1)
	line := []byte("KI" + long + " 0 0 0 \n")
	_, err := Read(bufio.NewReader(bytes.NewReader(line)))
	if err == nil {
		t.Fatal("overlong integer field should be rejected")
	}
}
