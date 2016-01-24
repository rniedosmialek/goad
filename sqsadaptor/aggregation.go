package sqsadaptor

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

// AggData type
type AggData struct {
	TotalReqs      int            `json:"total-reqs"`
	TotalTimedOut  int            `json:"total-timed-out"`
	AveTimeToFirst int64          `json:"ave-time-to-first"`
	TotBytesRead   int            `json:"tot-bytes-read"`
	Statuses       map[string]int `json:"statuses"`
	AveTimeForReq  int64          `json:"ave-time-for-req"`
	AveReqPerSec   int            `json:"ave-req-per-sec"`
	Slowest        int64          `json:"slowest"`
	Fastest        int64          `json:"fastest"`
	Region         string         `json:"region"`
}

// RegionsAggData type
type RegionsAggData struct {
	Regions map[string]AggData
}

func addResult(data *AggData, result *AggData) {
	initialDataTot := data.TotalReqs
	initialDataTot64 := int64(data.TotalReqs)
	data.TotalReqs += result.TotalReqs
	data.TotalTimedOut += result.TotalTimedOut
	dataTot64 := int64(data.TotalReqs)
	resultTot64 := int64(result.TotalReqs)
	data.AveTimeToFirst = (data.AveTimeToFirst*initialDataTot64 + result.AveTimeToFirst*resultTot64) / dataTot64
	data.TotBytesRead += result.TotBytesRead

	for key, value := range result.Statuses {
		data.Statuses[key] += value
	}
	data.AveTimeForReq = (data.AveTimeForReq*initialDataTot64 + result.AveTimeForReq*resultTot64) / dataTot64
	data.AveReqPerSec = (data.AveReqPerSec*initialDataTot + result.AveReqPerSec*result.TotalReqs) / data.TotalReqs
	if result.Slowest > data.Slowest {
		data.Slowest = result.Slowest
	}
	if result.Fastest < data.Fastest {
		data.Fastest = result.Fastest
	}
}

// Aggregate listens for results and sends totals, closing the channel when done
func Aggregate(outChan chan RegionsAggData, awsConfig *aws.Config, queueURL string, totalExpectedRequests uint) {
	defer close(outChan)
	data := RegionsAggData{make(map[string]AggData)}

	adaptor := NewSQSAdaptor(awsConfig, queueURL)
	timeoutStart := time.Now()
	for {
		result := adaptor.Receive()
		if result != nil {
			regionData, ok := data.Regions[result.Region]
			if !ok {
				regionData.Statuses = make(map[string]int)
				regionData.Region = result.Region
			}
			addResult(&regionData, result)
			data.Regions[result.Region] = regionData
			outChan <- data
			timeoutStart = time.Now()
		} else {
			waited := time.Since(timeoutStart)
			if waited.Seconds() > 20 {
				break
			}
		}
	}
}
