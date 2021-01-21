// Copyright 2014 Roger Peppe.
// See LICENCE file for details.

package errors_test

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/errgo.v2/fmt/errors"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

type errorsSuite struct{}

var _ = gc.Suite(&errorsSuite{})

func (*errorsSuite) TestNew(c *gc.C) {
	err := errors.New("foo") //err TestNew
	checkErr(c, err, nil, "foo", `[
	{$TestNew$: foo}
]`, err)
}

func (*errorsSuite) TestNewf(c *gc.C) {
	err := errors.Newf("foo %d", 99) //err TestNewf
	checkErr(c, err, nil, "foo 99", `[
	{$TestNewf$: foo 99}
]`, err)
}

var someErr = errors.New("some error") //err varSomeErr

func annotate1() error {
	err := errors.Note(someErr, nil, "annotate1") //err annotate1
	return err
}

func annotate2() error {
	err := annotate1()
	err = errors.Note(err, nil, "annotate2") //err annotate2
	return err
}

func (*errorsSuite) TestNoteUsage(c *gc.C) {
	err0 := annotate2()
	err, ok := err0.(errors.Wrapper)
	c.Assert(ok, gc.Equals, true)
	underlying := err.Underlying()
	checkErr(
		c, err0, underlying,
		"annotate2: annotate1: some error",
		`[
	{$annotate2$: annotate2}
	{$annotate1$: annotate1}
	{$varSomeErr$: some error}
]`,
		err0)
}

func (*errorsSuite) TestWrap(c *gc.C) {
	err0 := errors.Because(nil, someErr, "foo") //err TestWrap#0
	err := errors.Wrap(err0)                    //err TestWrap#1
	checkErr(c, err, err0, "foo", `[
	{$TestWrap#1$: }
	{$TestWrap#0$: foo}
]`, err)

	err = errors.Wrap(nil)
	c.Assert(err, gc.Equals, nil)
}

func (*errorsSuite) TestNoteWithNilError(c *gc.C) {
	c.Assert(errors.Note(nil, nil, "annotation"), gc.Equals, nil)
}

func (*errorsSuite) TestNote(c *gc.C) {
	err0 := errors.Because(nil, someErr, "foo") //err TestNote#0
	err := errors.Note(err0, nil, "bar")        //err TestNote#1
	checkErr(c, err, err0, "bar: foo", `[
	{$TestNote#1$: bar}
	{$TestNote#0$: foo}
]`, err)

	err = errors.Note(err0, errors.Is(someErr), "bar") //err TestNote#2
	checkErr(c, err, err0, "bar: foo", `[
	{$TestNote#2$: bar}
	{$TestNote#0$: foo}
]`, someErr)

	err = errors.Note(err0, func(error) bool { return false }, "") //err TestNote#3
	checkErr(c, err, err0, "foo", `[
	{$TestNote#3$: }
	{$TestNote#0$: foo}
]`, err)
}

func (*errorsSuite) TestNotef(c *gc.C) {
	err0 := errors.Because(nil, someErr, "foo")  //err TestNotef#0
	err := errors.Notef(err0, nil, "bar %d", 99) //err TestNotef#1
	checkErr(c, err, err0, "bar 99: foo", `[
	{$TestNotef#1$: bar 99}
	{$TestNotef#0$: foo}
]`, err)
}

func (*errorsSuite) TestCause(c *gc.C) {
	c.Assert(errors.Cause(someErr), gc.Equals, someErr)

	causeErr := errors.New("cause error")
	underlyingErr := errors.New("underlying error")          //err TestCause#1
	err := errors.Because(underlyingErr, causeErr, "foo 99") //err TestCause#2
	c.Assert(errors.Cause(err), gc.Equals, causeErr)

	checkErr(c, err, underlyingErr, "foo 99: underlying error", `[
	{$TestCause#2$: foo 99}
	{$TestCause#1$: underlying error}
]`, causeErr)

	err = customError{err}
	c.Assert(errors.Cause(err), gc.Equals, causeErr)
}

func (*errorsSuite) TestBecausef(c *gc.C) {
	c.Assert(errors.Cause(someErr), gc.Equals, someErr)

	causeErr := errors.New("cause error")
	underlyingErr := errors.New("underlying error")               //err TestBecausef#1
	err := errors.Becausef(underlyingErr, causeErr, "foo %d", 99) //err TestBecausef#2
	c.Assert(errors.Cause(err), gc.Equals, causeErr)

	checkErr(c, err, underlyingErr, "foo 99: underlying error", `[
	{$TestBecausef#2$: foo 99}
	{$TestBecausef#1$: underlying error}
]`, causeErr)
}

func (*errorsSuite) TestBecauseWithNoMessage(c *gc.C) {
	cause := errors.New("cause")
	err := errors.Because(nil, cause, "")
	c.Assert(err, gc.ErrorMatches, "cause")
	c.Assert(errors.Cause(err), gc.Equals, cause)
}

func (*errorsSuite) TestBecauseWithUnderlyingButNoMessage(c *gc.C) {
	err := errors.New("something")
	cause := errors.New("cause")
	err = errors.Because(err, cause, "")
	c.Assert(err, gc.ErrorMatches, "something")
	c.Assert(errors.Cause(err), gc.Equals, cause)
}

func (*errorsSuite) TestBecauseWithAllZeroArgs(c *gc.C) {
	err := errors.Because(nil, nil, "")
	c.Assert(err, gc.Equals, nil)
}

func (*errorsSuite) TestDetails(c *gc.C) {
	c.Assert(errors.Details(nil), gc.Equals, "[]")

	otherErr := fmt.Errorf("other")
	checkErr(c, otherErr, nil, "other", `[
	{other}
]`, otherErr)

	err0 := customError{errors.New("foo")} //err TestDetails#0
	checkErr(c, err0, nil, "foo", `[
	{$TestDetails#0$: foo}
]`, err0)

	err1 := customError{errors.Note(err0, nil, "bar")} //err TestDetails#1
	checkErr(c, err1, err0, "bar: foo", `[
	{$TestDetails#1$: bar}
	{$TestDetails#0$: foo}
]`, err1)

	err2 := errors.Wrap(err1) //err TestDetails#2
	checkErr(c, err2, err1, "bar: foo", `[
	{$TestDetails#2$: }
	{$TestDetails#1$: bar}
	{$TestDetails#0$: foo}
]`, err2)
}

func (*errorsSuite) TestSetLocation(c *gc.C) {
	err := customNewError() //err TestSetLocation#0
	checkErr(c, err, nil, "custom", `[
	{$TestSetLocation#0$: custom}
]`, err)
}

func customNewError() error {
	err := errors.New("custom")
	errors.SetLocation(err, 1)
	return err
}

func checkErr(c *gc.C, err, underlying error, msg string, details string, cause error) {
	c.Assert(err, gc.NotNil)
	c.Assert(err.Error(), gc.Equals, msg)
	if err, ok := err.(errors.Wrapper); ok {
		c.Assert(err.Underlying(), gc.Equals, underlying)
	} else {
		c.Assert(underlying, gc.Equals, nil)
	}
	c.Assert(errors.Cause(err), gc.Equals, cause)
	wantDetails := replaceLocations(details)
	c.Assert(errors.Details(err), gc.Equals, wantDetails)
}

func replaceLocations(s string) string {
	t := ""
	for {
		i := strings.Index(s, "$")
		if i == -1 {
			break
		}
		t += s[0:i]
		s = s[i+1:]
		i = strings.Index(s, "$")
		if i == -1 {
			panic("no second $")
		}
		file, line := location(s[0:i])
		t += fmt.Sprintf("%s:%d", file, line)
		s = s[i+1:]
	}
	t += s
	return t
}

func location(tag string) (string, int) {
	line, ok := tagToLine[tag]
	if !ok {
		panic(fmt.Errorf("tag %q not found", tag))
	}
	return filename, line
}

var tagToLine = make(map[string]int)
var filename string

func init() {
	data, err := ioutil.ReadFile("errors_test.go")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if j := strings.Index(line, "//err "); j >= 0 {
			tagToLine[line[j+len("//err "):]] = i + 1
		}
	}
	_, filename, _, _ = runtime.Caller(0)
}

type customError struct {
	error
}

func (e customError) Location() (string, int) {
	if err, ok := e.error.(errors.Locator); ok {
		return err.Location()
	}
	return "", 0
}

func (e customError) Underlying() error {
	if err, ok := e.error.(errors.Wrapper); ok {
		return err.Underlying()
	}
	return nil
}

func (e customError) Message() string {
	if err, ok := e.error.(errors.Wrapper); ok {
		return err.Message()
	}
	return ""
}

func (e customError) Cause() error {
	if err, ok := e.error.(errors.Causer); ok {
		return err.Cause()
	}
	return nil
}
