package main

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"gorm.io/gorm/logger"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	. "gopkg.in/telebot.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config struct for toml config file
type Config struct {
	Bot           *Bot
	Token         string
	SendToGroupID int64 `json:"id"`
	SendToGroup   Chat
	UseMysql      string
	MysqlConfig   string
	StartMessage  string
	HelpMessage   string
	GroupMessage  string
	HealthMessage string
	AdminID       string
}

type MysqlApp struct {
	CreatedAt time.Time
	UserName  string `gorm:"column:user_name"`
	UserID    int64  `gorm:"column:user_id"`
	GroupID   int64  `gorm:"column:group_id"`
	GroupName string `gorm:"column:group_name"`
	Message   string `gorm:"column:message;size:5000"`
}

var (
	config      Config
	bot         *Bot
	logTemplate string
	err         error
	db          *gorm.DB
	m           MysqlApp
)

func init() {
	config.Token = getEnvDefault("TOKEN", "")

	if err = testToken(); err != nil {
		log.Fatal(err)
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	time.Local = loc

	if n, err := strconv.Atoi(getEnvDefault("SEND_TO_GROUP_ID", "")); err == nil {
		config.SendToGroupID = int64(n)
	} else {
		log.Printf("SEND_TO_GROUP_ID:[%v] is not an integer.", n)
	}

	config.SendToGroup = Chat{
		ID:   config.SendToGroupID,
		Type: "group",
	}
	config.UseMysql = getEnvDefault("USE_MYSQL", "no")
	config.MysqlConfig = getEnvDefault("MYSQL_CONFIG", "user:name@tcp(ip:port)/database_name?charset=utf8mb4&parseTime=True&loc=Local")

	if os.Getenv("NO_PROXY") != "" || os.Getenv("no_proxy") != "" {
		log.Printf("NO_PROXY")
	} else {
		if os.Getenv("HTTP_PROXY") != "" {
			log.Printf("HTTP_PROXY: %v ", os.Getenv("HTTP_PROXY"))
		}
		if os.Getenv("HTTPS_PROXY") != "" {
			log.Printf("HTTPS_PROXY: %v ", os.Getenv("HTTPS_PROXY"))
		}
		if os.Getenv("ALL_PROXY") != "" {
			log.Printf("ALL_PROXY: %v ", os.Getenv("ALL_PROXY"))
		}
		if os.Getenv("http_proxy") != "" {
			log.Printf("http_proxy: %v ", os.Getenv("http_proxy"))
		}
		if os.Getenv("https_proxy") != "" {
			log.Printf("https_proxy: %v ", os.Getenv("https_proxy"))
		}
		if os.Getenv("all_proxy") != "" {
			log.Printf("all_proxy: %v ", os.Getenv("all_proxy"))
		}
	}

	if config.UseMysql == "yes" {
		log.Printf("Mysql Enable!")
		db, err = gorm.Open(mysql.New(mysql.Config{
			DSN:                       config.MysqlConfig, // DSN data source name
			DefaultStringSize:         256,                // string 类型字段的默认长度
			DisableDatetimePrecision:  true,               // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
			DontSupportRenameIndex:    true,               // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
			DontSupportRenameColumn:   true,               // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
			SkipInitializeWithVersion: false,              // 根据当前 MySQL 版本自动配置
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err = db.AutoMigrate(&MysqlApp{}); err != nil {
			log.Fatalf("自动创建表失败, %s", err)
			return
		}
	}
	config.StartMessage = getEnvDefault("START_MESSAGE", "welcome!")
	config.HelpMessage = getEnvDefault("HELP_MESSAGE", "help")
	config.HealthMessage = getEnvDefault("HEALTH_MESSAGE", "health")
	config.GroupMessage = getEnvDefault("GROUP_MESSAGE", "group")
	config.AdminID = getEnvDefault("ADMIN_ID", "")
}

func main() {
	httpClient := privacyDns()

	bot, err = NewBot(Settings{
		Token:  config.Token,
		Poller: &LongPoller{Timeout: 10 * time.Second},
		Client: httpClient,
	})
	if err != nil {
		log.Fatalf("Cannot start bot. Error: %v\n", err)
	}

	bot.Handle("/start", func(c Context) error {
		tgLog(c, "group")
		if _, err := bot.Send(c.Chat(), config.StartMessage+"\n"+config.HelpMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/health", func(c Context) error {
		tgLog(c, "health")
		if _, err := bot.Send(c.Chat(), config.HealthMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/help", func(c Context) error {
		tgLog(c, "help")
		if _, err := bot.Send(c.Chat(), config.HelpMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/group", func(c Context) error {
		tgLog(c, "group")
		if _, err := bot.Send(c.Chat(), config.GroupMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/exit", func(c Context) error {
		tgLog(c, "exit")
		if config.AdminID != "" && strconv.FormatInt(c.Sender().ID, 10) != config.AdminID {
			if _, err := bot.Send(c.Chat(), "你没有权限执行此操作!"); err != nil {
				log.Println(err)
			}
			return err
		}

		if _, err := bot.Send(c.Chat(), "Bot 即将停止服务!"); err != nil {
			log.Println(err)
		}
		os.Exit(1)
		return err
	})
	bot.Handle(OnText, func(c Context) error {
		tgLog(c, "forward")
		if ok, _ := regexp.MatchString("^(\\d+)$", c.Text()); ok {
			if _, err := bot.Send(c.Chat(), "Bot 不提供查询功能"); err != nil {
				log.Println(err)
			}
			return err
		}
		if _, err := bot.Send(&config.SendToGroup, c.Text()); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle(OnMedia, func(c Context) error {
		tgLog(c, "forward media")
		if _, err = bot.Send(&config.SendToGroup, c.Text()+"\n\n附件无法发送"); err != nil {
			log.Println(err)
		}
		return err
	})
	log.Println("Bot started!")
	bot.Start()
	//go func() {
	//	bot.Start()
	//}()
	//
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	//<-signalChan
	//log.Println("Shutdown signal received, exiting...")
}

func getEnvDefault(key, defVal string) string {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	return val
}

func testToken() (err error) {
	if config.Token == "" {
		err = errors.Errorf("Env variable %v isn't set!", config.Token)
		return err
	}
	match, err := regexp.MatchString(`^[0-9]+:.*$`, config.Token)
	if err != nil {
		return err
	}
	if !match {
		err = errors.Errorf("Telegram Bot Token [%v] is incorrect. Token doesn't comply with regexp: `^[0-9]+:.*$`. Please, provide a correct Telegram Bot Token through env variable TGTOKEN", config.Token)
		return err
	}
	return err
}

func tgLog(c Context, msg string) {
	logTemplate = fmt.Sprintf("user:{%v %v %v} in chat:[%v]", c.Sender().FirstName, c.Sender().LastName, c.Chat().ID, c.Text())
	m.UserName = c.Sender().FirstName + " " + c.Sender().LastName
	m.UserID = c.Chat().ID
	m.Message = c.Text()
	if c.Chat().Type == "group" {
		logTemplate = fmt.Sprintf("user:{%v %v %v} group:{%v %v} in chat:[%v]", c.Sender().FirstName, c.Sender().LastName, c.Sender().ID, c.Chat().Title, c.Chat().ID, c.Text())
		m.GroupName = c.Chat().Title
		m.GroupID = c.Chat().ID
	}
	log.Printf("%v request from %v", msg, logTemplate)
	if config.UseMysql == "yes" {
		if err = db.Create(&m).Error; err != nil {
			log.Fatalf("写入数据失败, %s", err)
			return
		}
	}
}

func privacyDns() (client *http.Client) {
	// 设置制定DNS 保护隐私
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(5000) * time.Millisecond,
				}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
	client = &http.Client{
		Timeout: 50 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
			DialContext:     dialContext,
		},
	}
	return client
}
