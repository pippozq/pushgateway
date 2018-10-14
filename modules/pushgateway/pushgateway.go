package pushgateway

import (
	"encoding/json"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/pippozq/pushgateway/constants/errors"
	"github.com/pippozq/pushgateway/global"
	"github.com/pippozq/pushgateway/modules/redis"
	"github.com/sirupsen/logrus"
	"runtime"
	"strconv"
	"strings"
)

type PushGateWay struct {
	data  *PushData
	Agent *redis.Agent
}

func NewPushGateWayController(data *PushData, agent *redis.Agent) *PushGateWay {
	return &PushGateWay{
		data:  data,
		Agent: agent,
	}
}

func (p *PushGateWay) getText(data PushData) (metricText string) {
	for _, metric := range data.Metrics {
		var labelList []string
		for labelKey, labelValue := range metric.Labels {
			t := fmt.Sprintf("%s=\"%s\"", labelKey, labelValue)
			labelList = append(labelList, t)
		}

		metricText += fmt.Sprintf("%s{%s} %f\n", metric.MetricName, strings.Join(labelList, ","), metric.MetricValue)
	}
	return
}

type Metric struct {
	MetricName  string            `json:"metric_name"`            // metric name, like test_metric
	MetricValue float64           `json:"metric_value,omitempty"` // metric value
	Labels      map[string]string `json:"labels"`                 // Label
}

type PushData struct {
	Metrics    []Metric `json:"metrics"`
	JobName    string   `json:"job_name"`              // Job Name
	ID         string   `json:"id"`                    // Id
	ExpireTime int      `json:"expire_time,omitempty"` // metric expire time，default 1800 seconds
}

type RespData struct {
	Status       string `json:"status"`
	Code         int    `json:"code"`
	ErrorMessage string `json:"error_message"`
}

func (p *PushGateWay) CacheMetric() (respData *RespData, err error) {
	expireTime := global.Config.RedisAgent.RedisExpireTime

	if p.data.ExpireTime > 0 {
		expireTime = strconv.Itoa(p.data.ExpireTime)
	}

	pushData, err := json.Marshal(p.data)
	if err != nil {
		return nil, err
	}
	p.Agent.Set(fmt.Sprintf("%s_%s", p.data.JobName, p.data.ID), pushData, expireTime)
	respData = new(RespData)
	respData.Code = 200
	return respData, nil
}

func (p *PushGateWay) getMetricList(key string, metricList []*PushData) {
	metricsByte, err := p.Agent.Get(key)
	if err != nil {
		logrus.Error(err)
	}
	metric := new(PushData)
	err = json.Unmarshal(metricsByte, &metric)
	if err != nil {
		logrus.Error(err)
	}
	metricList = append(metricList, metric)
}

func (p *PushGateWay) GetMetrics() (metricByte []byte, err error) {
	var metricList []*PushData
	keyList, err := p.Agent.GetKeyList("*")
	if err != nil {
		return nil, errors.MetricNotFound
	}

	metricPool := tunny.NewFunc(runtime.NumCPU(), func(payload interface{}) interface{} {
		key := payload.(string)
		metricsByte, err := p.Agent.Get(key)
		if err != nil {
			logrus.Error(err)
		}
		metric := new(PushData)
		err = json.Unmarshal(metricsByte, &metric)
		if err != nil {
			logrus.Error(err)
		}
		metricList = append(metricList, metric)

		return metricList
	})
	defer metricPool.Close()

	metricPool.SetSize(global.Config.PoolSize)

	for _, key := range keyList {
		metricPool.Process(key)
	}
	metricStr := ""
	for _, metric := range metricList {
		metricStr += p.getText(*metric)
	}

	return []byte(metricStr), nil
}

func (p *PushGateWay) GetMetric(jobName, id string) (metric *PushData, err error) {
	searchKey := fmt.Sprintf("%s_%s", jobName, id)

	keyList, err := p.Agent.GetKeyList("*")
	if err != nil {
		return nil, err
	}
	exist := false
	for _, key := range keyList {
		if searchKey == key {
			exist = true
		}
	}

	if !exist {
		return nil, errors.MetricNotFound
	}

	metricByte, err := p.Agent.Get(searchKey)

	metric = new(PushData)

	err = json.Unmarshal(metricByte, &metric)

	if err != nil {
		return nil, err
	}

	return metric, nil
}
