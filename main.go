package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 配置文件字段
type Configuration struct {
	ElasticSearch ElasticSearchType `mapstructure:"ElasticSearch",`
	Deadline      DeadlineType      `mapstructure:"Dadline"`
	Log           LogType           `mapstructure:"Log"`
}

type ElasticSearchType struct {
	// ES客户端地址
	ES_CLIENT_HOST string `mapstructure:"ES_CLIENT_HOST"`
	// es用户
	USERNAME string `mapstructure:"USERNAME"`
	// es密码
	PASSWORD string `mapstructure:"PASSWORD"`
	// 获取接口
	GETDATA_URL string `mapstructure:"GETDATA_URL"`
	// 基础默认索引 跳过默认索引
	DEFAULT_INDEX string `mapstructure:"DEFAULT_INDEX"`
}

type DeadlineType struct {
	// 日志过期时间
	DETELINE    int64  `mapstructure:"DETELINEX"`
	CHECK_CROND string `mapstructure:"CHECK_CROND"`
}

type LogType struct {
	FilePath string `mapstructure:"FilePath"`
}

// 配置对象
var Config Configuration

// 日志对象
var Lg *zap.Logger

// 检查是否存在默认索引
func DefaultIndexCheck(index string) bool {
	i := strings.Split(Config.ElasticSearch.DEFAULT_INDEX, ",")
	for _, v := range i {
		if index == v {
			return true
		}
	}
	return false
}

// 基于过期时间删除索引
func Delete_index(index string) (bool, string) {
	// 删除触发
	url := Config.ElasticSearch.ES_CLIENT_HOST + "/" + index
	req, _ := http.NewRequest("DELETE", url, nil)
	req.SetBasicAuth(Config.ElasticSearch.USERNAME, Config.ElasticSearch.PASSWORD)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return false, err.Error()
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return false, err.Error()
	}

	// 判断是否删除成功
	if response.StatusCode == 200 {
		fmt.Println(string(data))
		return true, "success"
	} else if response.StatusCode == 401 {
		return false, "认证失败"
	} else if response.StatusCode == 404 {
		return false, "索引不存在"
	} else {
		return false, err.Error()
	}
}

func InitLog() {
	// log config
	// 实例化zap配置
	cfg := zap.NewDevelopmentConfig()
	// 配置日志输出的地址
	cfg.OutputPaths = []string{
		fmt.Sprintf("%ses_drop_%s.log", Config.Log.FilePath, GetNowFormatTodayTime()),
		"stdout",
	}
	// 创建logger实例
	logg, _ := cfg.Build()
	zap.ReplaceGlobals(logg)
	Lg = logg
}

// 日志格式化
func GetNowFormatTodayTime() string {
	now := time.Now()
	dateStr := fmt.Sprintf("%02d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
	return dateStr
}

// 配置初始化
func InitConfig() {
	// 使用viper来对配置进行实例化
	v := viper.New()
	v.SetConfigFile("/app/config.yaml")
	// v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		color.Red("读取配置文件失败， 请检查配置是否正确！")
		panic(err)
	}
	serverConfig := Configuration{}
	if err := v.Unmarshal(&serverConfig); err != nil {
		color.Red("格式化文件失败， 请检查配置字段对接是否正确！")
		// panic(err)
	}

	Config = serverConfig
	color.Green("配置获取成功")
}

func Check_Index() {
	// 拼接索引地址
	url := Config.ElasticSearch.ES_CLIENT_HOST + Config.ElasticSearch.GETDATA_URL
	// 发起请求
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(Config.ElasticSearch.USERNAME, Config.ElasticSearch.PASSWORD)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		Lg.Error("request error")
	}
	body, _ := io.ReadAll(response.Body)
	data := strings.Split(string(body), "open")

	// 过滤数据是否符合规定时间
	for i, _ := range data {
		// fmt.Println(i, v)
		index := strings.Split(data[i], " ")
		index_name := index[3]
		ok := DefaultIndexCheck(index_name)

		if !ok {
			index, _ := regexp.MatchString("\\d{4}-\\d{1,2}-\\d{1,2}", index_name)
			if index {
				now := time.Now().Unix()
				endindex := strings.Split(index_name, "-")
				LenIndex := len(endindex)
				deadline := fmt.Sprintf("%s-%s-%s 00:00:00", endindex[LenIndex-3 : LenIndex][0], endindex[LenIndex-3 : LenIndex][1], endindex[LenIndex-3 : LenIndex][2])
				loc, _ := time.LoadLocation("Asia/Shanghai")
				tt, _ := time.ParseInLocation("2006-01-02 15:04:05", deadline, loc)
				endtime := (now - tt.Unix()) / 3600 / 24
				// 判断是否大于过期时间 进行删除操作
				if endtime > Config.Deadline.DETELINE {
					oks, msg := Delete_index(index_name)
					if oks {
						Lg.Info(fmt.Sprintf("index - %s", msg))
					} else {
						Lg.Info(fmt.Sprintf("index - %s", msg))
					}
				} else {
					Lg.Info(fmt.Sprintf("索引在可用期限内 - %s", index_name))
				}
			}
		}
	}
}

// 过滤特定时间内的索引
func main() {
	// 初始化配置
	InitConfig()
	// 日志初始化
	InitLog()

	// 开始计划任务
	c := cron.New()
	fmt.Println("计划任务： ", Config.Deadline.CHECK_CROND)
	c.AddFunc(Config.Deadline.CHECK_CROND, func() {
		Check_Index()
		Lg.Info("执行了计划任务")
	})
	c.Start()
	color.Green("计划任务启动成功")
	Lg.Info("计划任务启动成功")
	// 持久任务
	r := gin.Default()
	r.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"msg": "script is running",
		})
	})
	r.Run(":9090")
}
