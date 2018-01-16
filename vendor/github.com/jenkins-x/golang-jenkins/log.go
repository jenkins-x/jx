package gojenkins

import (
	"fmt"
	"net/url"
	"net/http"
	"strconv"
	"io/ioutil"
	"io"
	"time"
)

type LogData struct {
	Data     []byte
	TextSize int64
	MoreData bool
}

/*
// GetLog returns the log from the given start port (default to zero for all of it)
func (jenkins *Jenkins) GetLog(job Job, buildNumber int, start int64) (LogData, error) {
	return jenkins.GetLogFromURL(jenkins.GetBuildURL(job, buildNumber), start)
}
*/

func (jenkins *Jenkins) GetBuildURL(job Job, buildNumber int) string {
	return fmt.Sprintf("%s/%d", FullPath(job.FullName), buildNumber)
}

// GetLog returns the log from the given start port (default to zero for all of it)
func (jenkins *Jenkins) GetLogFromURL(buildUrl string, start int64, logData *LogData) error {
	path := fmt.Sprintf("%s/logText/progressiveText", buildUrl)
	params := url.Values{}
	params["start"] = []string{strconv.FormatInt(start, 10)}

	requestUrl := jenkins.buildUrlRaw(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return err
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//fmt.Printf("\n---------- headers %v\n", resp.Header)
	moreDataHeaders := resp.Header["X-More-Data"]
	sizeHeaders := resp.Header["X-Text-Size"]

	logData.MoreData = moreDataHeaders != nil && len(moreDataHeaders) > 0 && moreDataHeaders[0] == "true"

	if sizeHeaders != nil && len(sizeHeaders) > 0 {
		sh := sizeHeaders[0]
		if len(sh) > 0 {
			i, err := strconv.ParseInt(sh, 10, 64)
			if err == nil && i >= 0 {
				logData.TextSize = i
			}
		}
	}
	if resp.StatusCode != 302 && resp.StatusCode != 405 && resp.StatusCode > 201 {
		return fmt.Errorf("Invalid response %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logData.Data = data
	return nil
}

// TailLog tails the jenkins log to the given writer polling until there is no more output
// if timeoutPeriod is specified as a positive value then the tail will terminate if the timeout is reached before the log is completely
// fetched
func (jenkins *Jenkins) TailLog(buildUrl string, writer io.Writer, pollTime time.Duration, timeoutPeriod time.Duration) error {
	poller := jenkins.NewLogPoller(buildUrl, writer)
	fn := func() (bool, error) {
		return poller.Apply()
	}
	return Poll(pollTime, timeoutPeriod, fmt.Sprintf("to get all of the log of job %s", buildUrl), fn)
}

type LogPoller struct {
	jenkins  *Jenkins
	buildUrl string
	writer   io.Writer
	start    int64
	logData  LogData
}

// TailLogFunc returns a ConditionFunc for polling the log on the given URL to the given writer
func (jenkins *Jenkins) TailLogFunc(buildUrl string, writer io.Writer) ConditionFunc {
	poller := jenkins.NewLogPoller(buildUrl, writer)
	return func() (bool, error) {
		return poller.Apply()
	}
}

// NewLogPoller returns a LogPoller for polling the log on the given URL to the given writer
func (jenkins *Jenkins) NewLogPoller(buildUrl string, writer io.Writer) *LogPoller {
	return &LogPoller{
		jenkins:  jenkins,
		buildUrl: buildUrl,
		writer:   writer,
	}
}

func (p *LogPoller) Apply() (bool, error) {
	//fmt.Printf("Loading jenkins log from %d\n", start)
	err := p.jenkins.GetLogFromURL(p.buildUrl, p.start, &p.logData)
	if err != nil {
		return false, err
	}
	data := p.logData.Data
	if data != nil && len(data) > 0 {
		p.writer.Write(data)
	}
	if !p.logData.MoreData {
		return true, nil
	}
	p.start = p.logData.TextSize
	return false, nil
}
