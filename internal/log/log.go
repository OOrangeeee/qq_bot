package log

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

var Log *logrus.Logger

type CustomFormatter struct {
	TimestampFormat string
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	entry.Time = entry.Time.In(loc)
	// 创建一个 map 来构建想要的 JSON 结构
	data := map[string]interface{}{
		"time":    entry.Time.Format(f.TimestampFormat),
		"level":   entry.Level.String(),
		"message": entry.Message,
		"caller":  entry.Caller,
	}
	if len(entry.Data) > 0 {
		data["fields"] = entry.Data
	}
	// 序列化为 JSON
	serialized, err := json.MarshalIndent(data, "", "    ") // 美化输出
	if err != nil {
		return nil, err
	}
	return append(serialized, '\n'), nil
}

func InitLog() {
	Log = logrus.New()
	Log.SetFormatter(&CustomFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	Log.SetReportCaller(true)
	mw := io.MultiWriter(os.Stdout)
	Log.SetOutput(mw)
}
