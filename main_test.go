package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHDB struct {
	mock.Mock
}

type MockWindow mock.Mock
type MockInputField mock.Mock
type MockReader mock.Mock
type MockScanner struct {
	mock.Mock
	Called bool
}

func (m *MockScanner) Scan() bool {
	args := m.Called
	return args.Bool(0)
}

func TestMain(t *testing.T) {
	// db needs to be properly initialized and contain the history table. browserWin and UrlBar cannot be empty.
	// urlAddr needs to be a valid URL string. Check if we can write "http://example.com" to urlStr channel.
	// timestamp needs to be a valid time.Time object. Check if we can write time.Now() to timestamp
	// bufio.Scan needs to return something.
	// mock the db
	mockDB := new(MockHDB)
	mockDB.On("Add", "http://example.com", time.Now()).Return(nil)
	mockDB.On("Get", 0).Return(nil, nil)
	mockDB.On("GetAll").Return(nil, nil)
	mockDB.On("Delete", 0).Return(nil)
	mockDB.On("DeleteAll").Return(nil)
	mockDB.On("Count").Return(0, nil)
	mockDB.On("Connect", mock.Anything).Return(nil)
	mockDB.On("Close").Return(nil)

	// mock the browser window
	mockBrowserWin := new(MockWindow)
	mockBrowserWin.On("Clear").Return(nil)
	mockBrowserWin.On("Refresh").Return(nil)
	mockBrowserWin.On("SetCursor", 0, 0).Return(nil)
	mockBrowserWin.On("SetContent", mock.Anything).Return(nil)
	mockBrowserWin.On("SetOnKeyEvent", mock.Anything).Return(nil)
	mockBrowserWin.On("SetOnTextEvent", mock.Anything).Return(nil)

	// mock the url bar
	mockUrlBar := new(MockInputField)
	mockUrlBar.On("SetText", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnTextEvent", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnTextFinished", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnKeyEvent", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnMouseEvent", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnMouseFinished", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnSizeChanged", mock.Anything).Return(nil)
	mockUrlBar.On("SetOnClosing", mock.Anything).Return(nil)
	mockUrlBar.On("Show").Return(nil)
	mockUrlBar.On("Close").Return(nil)

	// mock the url string channel
	mockUrlStr := make(chan string)
	mockUrlStr <- "http://example.com"
	close(mockUrlStr)

	// mock the timestamp channel
	mockTimestamp := make(chan time.Time)
	mockTimestamp <- time.Now()
	close(mockTimestamp)

	// mock the bufio.Scan
	mockScanner := new(MockScanner)
	mockScanner.On("Scan").Return(true)
	mockScanner.On("Text").Return("http://example.com")

	// mock the bufio.Reader
	mockReader := new(MockReader)
	mockReader.On("ReadBytes", ' ').Return([]byte("http://example.com"), nil)

	// prepare assertions
	assert := assert.New(t)

	// run the main function
	main()
	var err error
	assert.Nil(err)
}
