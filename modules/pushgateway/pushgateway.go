package pushgateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/panjf2000/ants"
	"github.com/pippozq/pushgateway/constants/errors"
	"github.com/pippozq/pushgateway/global"
	"github.com/pippozq/pushgateway/modules/redis"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"text/template"
)

type PushGateWay struct {
	data           *PushData
	Agent          *redis.Agent
	MetricTemplate *template.Template
}

func NewPushGateWayController(data *PushData, agent *redis.Agent) *PushGateWay {
	t, err := template.ParseFiles("./config/label.template")
	if err != nil {
		logrus.Errorf("Read Template Error %s", err.Error())
		panic(err)
	}
	return &PushGateWay{
		data:           data,
		Agent:          agent,
		MetricTemplate: t,
	}
}

func (p *PushGateWay) getText(data PushData, textChan chan []string) {
	var metricList []string
	for _, metric := range data.Metrics {
		buff := bytes.NewBufferString("")
		err := p.MetricTemplate.ExecuteTemplate(buff, "label.template", metric)
		if err != nil {
			logrus.Error(err)
			continue
		}
		metricList = append(metricList, buff.String())
	}
	textChan <- metricList
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
		expireTime = p.data.ExpireTime
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

type PoolRunner struct {
	Key      string
	TextChan chan []string
}

func (p *PushGateWay) getMetricFromRedis(r interface{}) error {

	metricsByte, err := p.Agent.Get(r.(*PoolRunner).Key)
	if err != nil {
		logrus.Error(err)
		return err
	}
	metric := new(PushData)
	err = json.Unmarshal(metricsByte, &metric)
	if err != nil {
		logrus.Error(err)
		return err
	}
	p.getText(*metric, r.(*PoolRunner).TextChan)
	return nil
}

func (p *PushGateWay) GetMetrics() (metricByte []byte, err error) {
	keyList, err := p.Agent.GetKeyList("*")
	if err != nil {
		return nil, errors.MetricNotFound
	}
	if len(keyList) == 0 {
		return nil, nil
	}
	textChan := make(chan []string, len(keyList))

	wg := new(sync.WaitGroup)
	antPool, _ := ants.NewPoolWithFunc(global.Config.PoolSize, func(i interface{}) error {
		p.getMetricFromRedis(i)
		wg.Done()
		return nil
	})
	defer antPool.Release()
	for _, key := range keyList {
		wg.Add(1)
		r := &PoolRunner{
			Key:      key,
			TextChan: textChan,
		}
		antPool.Serve(r)
	}
	wg.Wait()

	var metricStrList []string

	for ml := range textChan {
		for _, metricValue := range ml {
			metricStrList = append(metricStrList, metricValue)
		}

		if len(textChan) <= 0 {
			close(textChan)
			break
		}
	}
	metricStr := strings.Join(metricStrList, "\n")
	metricStr = fmt.Sprintf("%s\n", metricStr)
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
