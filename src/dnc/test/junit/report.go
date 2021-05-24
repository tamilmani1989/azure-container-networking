package junit

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

// Adapted from github.com/jstemmer/go-junit-report

// TestSuites is a collection of JUnit test suites.
type TestSuites struct {
	XMLName xml.Name    `xml:"testsuites"`
	Suites  []TestSuite `xml:"testsuite"`
}

// TestSuite is a single JUnit test suite which may contain many testcases.
type TestSuite struct {
	XMLName    xml.Name   `xml:"testsuite"`
	Tests      int        `xml:"tests,attr"`
	Failures   int        `xml:"failures,attr"`
	Time       string     `xml:"time,attr"`
	Name       string     `xml:"name,attr"`
	Properties []Property `xml:"properties>property,omitempty"`
	TestCases  []TestCase `xml:"testcase"`
}

func (t *TestSuite) AddTestCases(tcs []TestCase) {
	for _, tc := range tcs {
		t.TestCases = append(t.TestCases, tc)
		t.Tests++
		if tc.Failure != nil {
			t.Failures++
		}
	}
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName     xml.Name     `xml:"testcase"`
	Classname   string       `xml:"classname,attr"`
	Name        string       `xml:"name,attr"`
	Time        string       `xml:"time,attr"`
	SkipMessage *SkipMessage `xml:"skipped,omitempty"`
	Failure     *Failure     `xml:"failure,omitempty"`
	Properties  []Property   `xml:"properties>property,omitempty"`
}

// SkipMessage contains the reason why a testcase was skipped.
type SkipMessage struct {
	Message string `xml:"message,attr"`
}

// Property represents a key/value pair used to define properties.
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Failure contains data related to a failed test.
type Failure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

func WriteReport(suites TestSuites, w io.Writer) error {
	bs, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)

	_, _ = writer.WriteString(xml.Header)
	_, _ = writer.Write(bs)
	_ = writer.WriteByte('\n')
	return writer.Flush()
}

func FmtDuration(d time.Duration) string {
	sec := d / time.Second
	d -= sec * time.Second
	millis := d / time.Millisecond

	return fmt.Sprintf("%d.%03d", sec, millis)
}
