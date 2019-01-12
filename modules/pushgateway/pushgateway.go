package pushgateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	goRedis "github.com/go-redis/redis"
	"github.com/panjf2000/ants"
	"github.com/pippozq/pushgateway/constants/errors"
	"github.com/pippozq/pushgateway/global"
	"github.com/pippozq/pushgateway/modules/redis"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	MetricPrefix = "metric"
)

var (
	MetricsChannel = make(chan *PushData, global.Config.RedisAgent.KeyCount)
)

func init() {
	p := NewPushGateWayController(global.Config.RedisAgent)
	go p.CacheMetrics()
}

type PushGateWay struct {
	data           *PushData
	Agent          *redis.Agent
	MetricTemplate *template.Template
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

func NewPushGateWayController(agent *redis.Agent) *PushGateWay {
	t, err := template.ParseFiles("./config/label.template")
	if err != nil {
		global.Config.Log.Errorf("Read Template Error %s", err.Error())
		panic(err)
	}
	return &PushGateWay{
		Agent:          agent,
		MetricTemplate: t,
	}
}

func (p *PushGateWay) Paging(keys []string) (keysGroup [][]string) {

	keysGroup = make([][]string, 0)

	step := len(keys) / p.Agent.KeyCount

	if len(keys) <= p.Agent.KeyCount {
		keysGroup = append(keysGroup, keys)
		return
	}
	head := 0
	tail := p.Agent.KeyCount
	for i := 0; i <= step; i++ {
		if tail < len(keys) {
			if i == 0 {
				keysGroup = append(keysGroup, keys[head:tail])
				head = tail
				tail += p.Agent.KeyCount
			} else if i == step {
				keysGroup = append(keysGroup, keys[tail:])
			} else {
				keysGroup = append(keysGroup, keys[head:tail])
				head = tail
				tail += p.Agent.KeyCount
			}
		} else {
			keysGroup = append(keysGroup, keys[head:])
		}
	}
	return
}

type PipelineData struct {
	Expire int
	Key    string
	Value  []byte
}

func (p *PushGateWay) WriteToRedisPipeline(metrics []*PushData) (err error) {

	pipelineList := make([]*PipelineData, 0)
	for _, data := range metrics {

		mkv := new(PipelineData)

		mkv.Expire = p.Agent.RedisExpireTime
		mkv.Key = fmt.Sprintf("%s_%s_%s", MetricPrefix, data.JobName, data.ID)

		if data.ExpireTime > 0 {
			mkv.Expire = data.ExpireTime
		} else {
			mkv.Expire = p.Agent.RedisExpireTime
		}
		pushData, err := json.Marshal(data)
		if err != nil {
			global.Config.Log.Error(err)
			continue
		}
		mkv.Value = pushData

		pipelineList = append(pipelineList, mkv)
	}

	pipeline := p.Agent.Pool.Pipeline()

	for _, mkv := range pipelineList {
		pipeline.Set(mkv.Key, mkv.Value, time.Duration(mkv.Expire)*time.Second)
	}
	result, err := pipeline.Exec()
	if err != nil {
		return err
	}
	for _, r := range result {
		global.Config.Log.Debug(r.Name(), r.Args(), r.Err())
	}
	return nil
}

func (p *PushGateWay) ReadFromRedisPipeline(keys []string) (metricValues [][]byte, err error) {

	pipeline := p.Agent.Pool.Pipeline()

	for _, key := range keys {
		pipeline.Get(key)
	}

	result, err := pipeline.Exec()
	if err != nil {
		return nil, err
	}
	for _, r := range result {
		result, err := r.(*goRedis.StringCmd).Result()
		if err != nil {
			global.Config.Log.Error(err)
			continue
		}
		metricValues = append(metricValues, []byte(result))
	}
	return metricValues, nil
}

func (p *PushGateWay) getText(data PushData, textChan chan []string) {
	var metricList []string
	for _, metric := range data.Metrics {
		buff := bytes.NewBufferString("")
		err := p.MetricTemplate.ExecuteTemplate(buff, "label.template", metric)
		if err != nil {
			global.Config.Log.Error(err)
			continue
		}
		metricList = append(metricList, buff.String())
	}
	textChan <- metricList
	return
}

func (p *PushGateWay) CacheMetrics() {
	metrics := make([]*PushData, 0)
	for {
		select {
		case metric := <-MetricsChannel:
			if len(metrics) == global.Config.RedisAgent.KeyCount {
				go func(data []*PushData) {
					p.WriteToRedisPipeline(data)
				}(metrics)
				metrics = make([]*PushData, 0)

			} else {
				metrics = append(metrics, metric)
			}
		case <-time.After(time.Second * time.Duration(p.Agent.PipelineWaitTime)):
			if len(metrics) != 0 {
				go func(data []*PushData) {
					p.WriteToRedisPipeline(data)
				}(metrics)
				metrics = make([]*PushData, 0)
			}
			global.Config.Log.Debug("Wait Input")

		}
	}
}

func (p *PushGateWay) CacheMetric(data *PushData) (respData *RespData, err error) {
	MetricsChannel <- data

	respData = new(RespData)
	respData.Code = 200
	respData.Status = "cached"
	return respData, nil
}

func (p *PushGateWay) getMetricList(key string, metricList []*PushData) {
	metricsByte, err := p.Agent.Get(key)
	if err != nil {
		global.Config.Log.Error(err)
	}
	metric := new(PushData)
	err = json.Unmarshal(metricsByte, &metric)
	if err != nil {
		global.Config.Log.Error(err)
	}
	metricList = append(metricList, metric)
}

type PoolRunner struct {
	Keys     []string
	TextChan chan []string
}

func (p *PushGateWay) getMetricFromRedis(r interface{}) error {

	metricsValues, err := p.ReadFromRedisPipeline(r.(*PoolRunner).Keys)
	if err != nil {
		global.Config.Log.Error(err)
		return nil
	}
	for _, metricsValue := range metricsValues {
		metric := new(PushData)
		err = json.Unmarshal(metricsValue, &metric)
		if err != nil {
			global.Config.Log.Error(err)
			return nil
		}
		p.getText(*metric, r.(*PoolRunner).TextChan)
	}

	return nil
}

func (p *PushGateWay) GetMetrics() (metricByte []byte, err error) {
	var metricList []string
	keyList, err := p.Agent.GetKeyList(fmt.Sprintf("%s_*", MetricPrefix))
	if err != nil {
		global.Config.Log.Error(err)
		return nil, err
	}

	if len(keyList) == 0 {
		return nil, errors.MetricNotFound
	}

	keysGroup := p.Paging(keyList)

	textChan := make(chan []string, len(keyList))

	wg := new(sync.WaitGroup)

	antPool, _ := ants.NewPoolWithFunc(p.Agent.PoolSize, func(i interface{}) {
		if err = p.getMetricFromRedis(i); err != nil {
			global.Config.Log.Error(err)
		}
		wg.Done()
	})
	defer antPool.Release()
	for _, keys := range keysGroup {
		wg.Add(1)
		r := &PoolRunner{
			Keys:     keys,
			TextChan: textChan,
		}
		if err = antPool.Serve(r); err != nil {
			global.Config.Log.Error(err)
		}
	}
	wg.Wait()

	for ml := range textChan {
		for _, metricValue := range ml {
			metricList = append(metricList, metricValue)
		}

		if len(textChan) <= 0 {
			close(textChan)
			break
		}
	}
	metricStr := strings.Join(metricList, "\n")
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
